from issue_triage.triage import IssueTriage
import os
import logging

PROJECT_CARD_ID = os.getenv('INPUT_NEEDS_TRIAGE_PROJECT_CARD_ID')
ISSUE_NUMBER = os.getenv('INPUT_ISSUE_NUMBER')
REPO = os.getenv('GITHUB_REPOSITORY')

assert PROJECT_CARD_ID, "Input PROJECT_CARD_ID not supplied."
assert ISSUE_NUMBER, "Input ISSUE_NUMBER not supplied."
assert os.getenv("INPUT_GITHUB_PERSONAL_ACCESS_TOKEN"), "Must supply input PERSONAL_ACCESS_TOKEN for Action to Run."

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO,
                      format=('%(levelname)s|%(asctime)s'
                            '|%(message)s|%(pathname)s|%(lineno)d|'),
                    datefmt='%Y-%m-%dT%H:%M:%S',
                    )
                    
    triager = IssueTriage()
    url = f"https://github.com/{REPO}/issues/{ISSUE_NUMBER}"
    logging.info(f'Triaging issue {ISSUE_NUMBER} - {url}')
    issue_info = triager.triage_issue(url)
    logging.info(issue_info)