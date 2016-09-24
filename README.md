# jbuild

## Testing.
### Unit Tests
To run unit tests, use something like:

`
go test github.com/jeshuam/jbuild/...
`

### Functional Tests
To run functional tests, use something like:

`
go test github.com/jeshuam/jbuild
`

To run functional tests with coverage, use:

`
go test -coverprofile=coverage.out -coverpkg=github.com/jeshuam/jbuild/... github.com/jeshuam/jbuild && go tool cover -html=coverage.out && rm coverage.out
`
