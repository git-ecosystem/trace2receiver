## Contributing


Hi there! We're thrilled that you'd like to contribute to this
project. Your help is essential for keeping it great.

Contributions to this project are
[released](https://help.github.com/articles/github-terms-of-service/#6-contributions-under-repository-license)
to the public under the
[project's open source license](./LICENSE).

Please note that this project is released with a
[Contributor Code of Conduct](./CODE_OF_CONDUCT.md).
By participating in this project you agree to abide by its terms.


## Prerequisites for running and testing code

These are one time installations required to be able to test your
changes locally as part of the pull request (PR) submission process.

1. Install Go [through download](https://go.dev/doc/install) | [through Homebrew](https://formulae.brew.sh/formula/go)
1. Clone this repository.
1. Build and test your changes in isolation within your `trace2receiver` component.
1. Submit your PR.

Of course the whole point of this is to run your `trace2receiver`
component within a custom collector and generate some data, so you
should also test it in that context.
See
[Building and Configuration](./Docs/README.md).

1. Create an
[OpenTelemetry Custom Collector](https://opentelemetry.io/docs/collector/)
using the builder tool in a new peer repository that references the
`trace2receiver` component.
1. The `go.mod` file in your collector should reference the public version
of the `trace2receiver` component.  You may need to use a
[`replace`](https://go.dev/doc/modules/gomod-ref#replace)
to redirect GO to your development version for testing.
1. Build and test your component changes running under a collector.

Your custom collector should not be included in your PR; just changes
to the `trace2receiver` component.


## Submitting a pull request

1. Clone the repository
1. Make sure the tests pass on your machine: `go test -v ./...`
1. Create a new branch: `git checkout -b my-branch-name`
1. Make your change, add tests, and make sure the tests still pass
1. Push to your fork and submit a pull request.
1. Pat yourself on the back and wait for your pull request to be reviewed and merged.

Here are a few things you can do that will increase the likelihood of
your pull request being accepted:

- Write tests.
- Keep your change as focused as possible. If there are multiple changes you would like to make that are not dependent upon each other, consider submitting them as separate pull requests.
- Write a [good commit message](https://github.blog/2022-06-30-write-better-commits-build-better-projects/).


## Resources

- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Using Pull Requests](https://help.github.com/articles/about-pull-requests/)
- [GitHub Help](https://help.github.com)
