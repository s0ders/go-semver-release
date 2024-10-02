# How it works

When executed, the program will go through the following steps:

1. Load the configuration
2. Load the GPG key, if any
3. Clone or open the repository depending on whether the program is executed in local or remote mode
4. Loop on every configured branch whether they are release or prerelease
5. Loop on every project, if the program is executed in monorepo mode
6. Fetch the latest semantic version tag if it exists
7. If a latest SemVer tag is found, fetch every commit newer than the tag otherwise fetch every commit
8. Loop on every commit and bump the SemVer according to the specified release rules
9. Tag the repository and sign it if a GPG key was passed
10. Push the tag to remote if executed in remote mode
