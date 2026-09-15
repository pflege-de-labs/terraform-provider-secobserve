# Either the numeric id or the exact username works.
terraform import secobserve_user.ci_bot 7
terraform import secobserve_user.ci_bot "ci-bot"
