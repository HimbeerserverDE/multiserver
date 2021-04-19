# Contributing
## Goal
The main goal of this project is to make server hopping possible and to give minetest servers control over it. It should support protocol versions >= 37.
The secondary goal is to provide a plugin API that should at least have the capabilities of the MT Lua API.
This will be much easier if anon5's MT module is expanded to support the full network protocol.
## Issues
Issues should be opened to
* report bugs
* request new features
* discuss new features before opening a pull request
### Bug reports
When reporting bugs please use the **latest version** of multiserver and Minetest 5.3.0+ and provide the following information:
* Minetest version
* Related log messages (MT and multiserver)
* Whether it is a client or a server issue
* (Optional) What might be causing the issue
Please test if the bug is caused by multiserver before submitting an issue.
### Feature requests
When requesting a feature be sure to describe it in detail. Provide all the information necessary to implement it if you can.
## Pull requests
Recommended way to contribute:
1. Fork this project
2. Open a new branch for each feature or fix
3. Merge upstream changes if there are any
4. Code
5. Open one pull request per feature/fix
### Features
You can open an issue to discuss a feature before starting to code. This way you won't waste your time implementing a feature that is never going to get merged.
### Merge probability
#### Instant rejection
* Proprietary software
* Incompatible changes
#### Low
* Fixes for issues with one or more assignees
* Adding new ways to achieve something that is already possible (unless it is made significantly easier)
* Code not formatted by gofmt
#### Medium
* New features
* Fixing minor bugs
* Changes that close issues
#### High
* Fixing bugs
* Updating dependencies
* Increasing code quality
