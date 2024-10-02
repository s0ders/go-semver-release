# Output

The `release` command output is JSON formatted so that it can easily be parsed.

If executed in non-verbose mode, no matter the scenario (e.g., no new release, new release, dry-run) the output, will always have the following keys (values are given for example), and the program will produce one of these output per branch parsed:

```json
{
    "new-release": true,
    "version": "1.2.3",
    "branch": "master",
    "project": "foo",
    "message": "new release found"
}
```

{% hint style="info" %}
The `project` key will only be present in an output if executed in monorepo mode. See the "Multiple projects in a single repository or "monorepo"" section for more information.
{% endhint %}
