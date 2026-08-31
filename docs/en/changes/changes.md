## 0.12.0

#### Features

#### Bugs
- Take `release-passed.sh`'s default version from `dist/dev` rather than from `Chart.yaml`. By the time a vote passes, `release.sh` has already opened the next-version PR and it is usually merged, so `Chart.yaml` holds the version *after* the one being released -- pressing Enter at the prompt tried to publish a candidate that does not exist. It failed, but several prompts later and with an svn path error rather than an explanation. The script now reads what is actually waiting in `dist/dev`, refuses to guess when there is more than one candidate, and checks the chosen version exists before asking whether the vote passed.

#### Chores
