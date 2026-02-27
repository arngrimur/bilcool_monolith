# testdb

```bash
go get -u gitlab.tooling.zimpler.net/shared/embedded-postgres/v2/pkg/testdb
```

Package for conveniently connecting to and migrating a database instance for tests.

Just call `db := testdb.NewEmbedded(t)` to get a fully migrated database. By default uses dbmate looking for migrations files
under migrations/ in the project's root directory. This can be configured by passing `WithMigrater(...)` to `NewEmbbedded`.

The project's root directory is defined by the `ProjectRoot` type. Depending on the value
of this type, the project root is either defined by the parent of the .git/ directory, or the
directory of the "go.mod" file. The default behaviour is to look for the programs associated "go.mod" file. This can be configured 
by passing `WithProjectRoot(...)` to `NewEmbedded`.

Writes to .testdbdata/ in the project's root directory. Each test gets it's own subfolder, potentially creating a lof of files
if you have a lot of individual tests calling `NewEmbedded`. In most cases this shouldn't be a problem. However, if it
turns out to be a problem, you can reduce the number of subfolders by sharing the database instance for each logical group of
code, like so:

```go
// MyServiceTests holds methods for each MyService subtest. This type allows
// passing dependencies for tests while still providing a convenient syntax when
// subtests are registered.
type MyServiceTests struct {
	db *sql.DB
}

func TestMyService(t *testing.T) {
	t.Parallel()

	db := testdb.NewEmbedded(t, testdb.WithProjectRoot(testdb.GoModule))
	tests := MyServiceTests{db}

	t.Run("createSomething", tests.createSomething)
	t.Run("querySomething", tests.querySomething)
}

func (ms *MyServiceTests) createSomething(t *testing.T) {
	// Tests creating something.
}

func (ms *MyServiceTests) querySomething(t *testing.T) {
	// Tests querying for something.
}
```

The postgres daemon and default migrater uses the `io.Writer` from calling `NewWriteLogger(t, &bytes.Buffer{})`. This respects the 
verbosity flag when running `go test` -- i.e. you only get verbose output logs when running `go test -v` or when a test fails.

## Debugging

For debugging purposes, you can connect to a database after a test has run. This is done in the following way. 

1. Go to the directory containing the embedded-postgres binary:

```bash
cd .testdbdata/<testname>
```

2. Start the embedded-postgres daemon:

```bash
./bin/pg_ctl start -w -D data --options="-p 6432"
```

3. Connect to the database:

```bash
psql "postgres://postgres:postgres@localhost:6432/<testname>"
```

4. Once you are done, stop the embedded-postgres daemon:

```bash
./bin/pg_ctl stop -D data
```

