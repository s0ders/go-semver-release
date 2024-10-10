# Output

## Command output

The `release` command output is JSON formatted so that it can easily be parsed.

If executed in non-verbose mode, the output, will always have the following keys (values are given for example), and the program will produce one of these output per branch and per project, if executed in monorepo mode:

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
{"new-release":true,"version":"2.1.1-rc","branch":"rc","message":"new release found"}
```

## GitHub Action output

🚧 This section is a work in progress 🚧