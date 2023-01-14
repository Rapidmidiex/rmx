<!-- # golang-template-repository

> 💣 &nbsp; _All Setup content in this file should be replaced with your project details post setup._

## Setup

A gh-pages site is automatically generated for you when you clone/fork this repository. To get started, you must configure gh-pages with a few easy clicks for the site to be published. You can then follow the pre-written site docs to familiarize yourself with this repository.

### Steps

🎛️ &nbsp; Configure gh-pages as per instructions [here](https://rog-golang-buddies.github.io/golang-template-repository/continuous-integration/mkdocs-material/#ci-setup).

🌐 &nbsp; Goto your site at `https://github.com/Rapidmidiex/rmx` (the link is also made available via **Environments** section in your Github repo).

✋ &nbsp; Take a moment to review the `Quickstart` guide before you get started. It has critical prerequisites.

🧐 &nbsp; Peruse the `Continuous integration` docs to get yourself upto speed.

> _Having trouble accessing your site? Access the template repository Quickstart and Continuous integration docs here_: <br>
> https://rog-golang-buddies.github.io/golang-template-repository

🚀 &nbsp; Go build something amazing!

---

<br>

> _The following section provides a sample README template sourced from https://www.makeareadme.com_ -->

# RMX

Jams with friends and strangers in realtime.

## Description

RMX allows you to play music people around the globe in near realtime.

## Installation

<!-- accessing the web ui -->

<!-- tui installation -->

## Usage

### Web

TODO

### TUI

TODO

## Develop

Clone this repo and grab all the necessary dependencies.

```bash
$ git clone git@github.com:Rapidmidiex/rmx.git
$ cd rmx
$ go mod tidy
```

### Environment Variables

```env
POSTGRES_URL=[postgres://user:password@host:5432/rmx?sslmode=disable]

# Optional
PORT=8080 # Default
```

### Start the Server

#### Shell

```
$ go run ./cmd/server
```

#### VS Code

Copy `env.example` to `.env.development` and fill in the variables.

```
$ cp env.example .env.development
```

Run "Start App Server" debugger configuration.

#### CLI

TODO

### Runing the tests

We do not use database mocks and test against a real database. By default [dockertest](https://github.com/ory/dockertest) is used to create and tear down a Postgres database for testing.

If you wish to use your own database, supply the `TEST_POSTGRES_URL` environment variable when running the tests.

```
$ go test ./...
```

## Contributing

When contributing to this repository, please first discuss the change you wish to make via issue,
Slack, or any other method with the owners of this repository before making a change.

Please note we have a code of conduct, please follow it in all your interactions with the project.

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a
   build.
1. Update the README.md with details of changes to the interface, this includes new environment
   variables, exposed ports, useful file locations and container parameters.
1. You may merge the Pull Request in once you have the sign-off of two other developers, or if you
   do not have permission to do that, you may request the second reviewer to merge it for you.

## Code of Conduct

### Our Pledge

In the interest of fostering an open and welcoming environment, we as
contributors and maintainers pledge to making participation in our project and
our community a harassment-free experience for everyone, regardless of age, body
size, disability, ethnicity, gender identity and expression, level of experience,
nationality, personal appearance, race, religion, or sexual identity and
orientation.

### Our Standards

Examples of behavior that contributes to creating a positive environment
include:

-   Using welcoming and inclusive language
-   Being respectful of differing viewpoints and experiences
-   Gracefully accepting constructive criticism
-   Focusing on what is best for the community
-   Showing empathy towards other community members

Examples of unacceptable behavior by participants include:

-   The use of sexualized language or imagery and unwelcome sexual attention or
    advances
-   Trolling, insulting/derogatory comments, and personal or political attacks
-   Public or private harassment
-   Publishing others' private information, such as a physical or electronic
    address, without explicit permission
-   Other conduct which could reasonably be considered inappropriate in a
    professional setting

### Attribution

This Code of Conduct is adapted from the [Contributor Covenant][homepage], version 1.4,
available at [http://contributor-covenant.org/version/1/4][version]

[homepage]: http://contributor-covenant.org
[version]: http://contributor-covenant.org/version/1/4/

## License

_For open source projects, say how it is licensed._
_Commonly used open-source licenses are [MIT](https://opensource.org/licenses/MIT) and [Apache](https://www.apache.org/licenses/LICENSE-2.0)._
