// When using EUC as application-default you may need to run
// gcloud auth application-default set-quota-project ${PROJECT}
// to set the billing project

package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
	"github.com/kubeflow/code-intelligence/Label_Microservice/go/cmd/automl/pkg/automl"
	"github.com/kubeflow/code-intelligence/Label_Microservice/go/cmd/automl/pkg/server"
	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func init() {
	// Add filename as one of the fields of the structured log message.
	filenameHook := filename.NewHook()
	filenameHook.Field = "filename"
	log.AddHook(filenameHook)

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(labelCmd)

	labelCmd.AddCommand(labelModelCmd)

	serveCmd.Flags().StringVarP(&options.project, "project", "", "issue-label-bot-dev", "Project to get AutoML models for")
	serveCmd.Flags().StringVarP(&options.location, "location", "", "us-central1", "Location to search for models")
	serveCmd.Flags().StringVarP(&options.name, "name", "", "kubeflow_issues_with_repo", "The display name of the model; only models with this name will be considered")
	serveCmd.Flags().StringVarP(&options.kptFile, "kptFile", "", "", "The path to the KptFile containing the setter.")
	serveCmd.Flags().StringVarP(&options.setterName, "setterName", "", "automl-model", "The name of the setter.")
	serveCmd.Flags().StringVarP(&options.port, "port", "", "80", "The port to serve on.")
	serveCmd.Flags().StringVarP(&options.retrainInterval, "retrainInterval", "", "12h", "The time between retraining; this can be a string using the formats accepted by Duration.parse.")

	serveCmd.MarkFlagRequired("kptFile")

	getCmd.Flags().StringVarP(&getOptions.name, "name", "", "", "The model to get.")
	getCmd.Flags().StringVarP(&getOptions.project, "project", "", "issue-label-bot-dev", "Project to get AutoML models for")
	getCmd.Flags().StringVarP(&getOptions.location, "location", "", "us-central1", "Location to search for models")
	getCmd.Flags().StringVarP(&getOptions.outputFile, "output", "", "", "(Optional) If supplied write the evaluation scores to this file in csv format.")
	getCmd.MarkFlagRequired("model")
}

type cliOptions struct {
	project    string
	location   string
	name       string
	kptFile    string
	setterName string
	port       string
	retrainInterval string
}

type getCmdOptions struct {
	name string
	project    string
	location   string
	outputFile string
}

var (
	options    = cliOptions{}
	getOptions = getCmdOptions{}
	rootCmd    = &cobra.Command{
		Short: "An automl model controller",
		Long:  `A controller to synchronize your automl model with your configs`,
	}

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start webserver.",
		Long:  `starts the controller`,
		Run: func(cmd *cobra.Command, args []string) {
			router := mux.NewRouter().StrictSlash(true)

			interval, err := time.ParseDuration(options.retrainInterval)

			if err != nil {
				log.Fatalf("Could not parse --retrainInterval=%v as a duration; error %v", options.retrainInterval, err)
				return
			}

			s := &server.Server{
				Project:    options.project,
				Location:   options.location,
				Name:       options.name,
				KptFile:    options.kptFile,
				SetterName: options.setterName,
				RetrainInterval: interval,
			}
			router.HandleFunc("/needsSync", s.NeedsSync)
			router.HandleFunc("/needsTrain", s.NeedsTrain)
			router.HandleFunc("/", s.Healthz)

			address := ":" + options.port
			log.Infof("Serving on %v", address)
			log.Fatal(http.ListenAndServe(address, router))
		},
	}

	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get the specified model.",
		Long:  `Get the specified model`,
		Run: func(cmd *cobra.Command, args []string) {

			name := fmt.Sprintf("projects/%v/locations/%v/models/%v", getOptions.project, getOptions.location, getOptions.name)
			model, err := automl.GetModel(name)

			if err != nil {
				log.Fatalf("Error getting model %v; error: %v", getOptions.name, err)
			}

			b, err := yaml.Marshal(model)

			if err != nil {
				log.Fatalf("Error marshiling the model to yaml %v; error: %v", getOptions.name, err)
			}

			fmt.Printf(string(b) + "\n")

			evaluation, err := automl.GetModelEvaluation(name, getOptions.outputFile)

			e, err := yaml.Marshal(evaluation)

			if err != nil {
				log.Fatalf("Error marshiling the evaluation to yaml %v; error: %v", getOptions.name, err)
			}

			fmt.Printf(string(e) + "\n")
		},
	}

	labelCmd = &cobra.Command{
		Use:   "label",
		Short: "Add labels.",
	}

	labelModelCmd = &cobra.Command{
		Use:   "models",
		Short: "Label the specified model.",
		Long:  `Label the specified model`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				log.Fatalf("Error usage is label models <model> label1=value1 label2=value2")
			}

			name := args[0]

			labels := map[string]string{}
			for _, p := range args[1:] {
				if b, err := regexp.MatchString(".*=.*", p); err != nil || !b {
					if err != nil {
						log.Fatalf("Could not parse label: %v; error %v", p, err)
					}
					log.Fatalf("%v doesn't match label pattern of labelName=labelValue", p)
				}

				pieces := strings.SplitN(p, "=", 2)

				labels[pieces[0]] = pieces[1]
			}

			if err := automl.LabelModel(name, labels); err != nil {
				log.Fatalf("Error labeling model: %+v", err)
			}

		},
	}
)

func main() {
	rootCmd.Execute()
}
