# Output

## Command output

The `release` command output is JSON formatted so that it can easily be parsed.

If executed in non-verbose mode, the output will always have the following keys (values are given for example), and the program will produce one of these outputs per branch and per project, if executed in monorepo mode:

```json
{
    "new-release": true,
    "version": "1.2.3",
    "branch": "master",
    "project": "foo",
    "message": "new release found"
}
```

> [!NOTE]
> The `project` key will only be present in an output if executed in monorepo mode. See [this section](configuration.md#monorepo) for more information.

Here is an example of an output where two branches were parsed, please note that there are two separate JSON which means that for this output to be parsed, it needs to be read line by line:

```json
{"new-release":true,"version":"1.2.2","branch":"main","message":"new release found"}
{"new-release":true,"version":"2.1.1-rc.1","branch":"rc","message":"new release found"}
```

## GitHub Action output
Though this tool is CI agnostic, it will try to detect if it is being executed on a GitHub Action runner.
If the program is in [monorepo ](configuration.md#monorepo)mode, three outputs will be generated per branch/project pair:
* `<BRANCH_NAME>_SEMVER`, the latest semantic version
* `<BRANCH_NAME>_NEW_RELEASE`, whether a new release was found or not
* `<BRANCH_NAME>_PROJECT`, the name of the project inside the monorepo

If not in monorepo mode, two outputs will be generated per branch:
* `<BRANCH_NAME>_SEMVER`, the latest semantic version
* `<BRANCH_NAME>_NEW_RELEASE`, whether a new release was found or not