## Coding Style

This code base generally follows the standard Go style outlined in [effective
Go], with further clarifications in Go wiki [review comments]. If in doubt,
strive to align with existing code.

[effective Go]: https://go.dev/doc/effective_go
[review comments]: https://go.dev/wiki/CodeReviewComments

## Commits

Each commit should contain a single logical change. Changes to test code should
_generally_ be part of the same commit that necessitated the changes, not in
their own commit. The exception is when adding new tests that apply to multiple
commits.

If a commit only contains changes to code touched by prior commits in the same
pull request (e.g. to address review comments), it should not exist -- the
changes should merged into the existing commit(s).

Commit messages should follow the [conventional commits] style.

Commit messages should primarily answer two questions: (1) what, at a high
level, is being achieved by this commit, and (2) why?

Do NOT list every file/symbol touched by the commit in the message (this is
easily seen in the commit diff).

Do NOT reference GitHub issues/pull requests in commit messages. Commit
messages are often viewed directly from git and should generally stand on their
own. Instead, summarise the issue being addressed in the commit message itself.
It _is_ OK to include to include reference to external sources, such as
relevant standards/specifications.

(note: on the other hand, you SHOULD include reference to relevant issues/other
pulls in the pull request description.)

Commits MUST be signed off (use --signoff flag when creating the commit).

[conventional commits]: https://www.conventionalcommits.org/en/v1.0.0/#summary

## Pre-submit checks

Once you have everything prepared, run

```bash
make presubmit
```

and make sure it completes without errors.
