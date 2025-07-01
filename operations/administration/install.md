# Installing Postgres on MacOS

The script below demonstrates how to install specific postgres version using ̱Homebrew.

> Prefer using Docker to run Postgres locally for testing.

```bash
$ brew install postgresql@9.6
```

# Multiple Postgres Running

Sometimes there is already an instance of Postgres running locally that conflicts with Postgres running in Docker. To check if port 5432 is already being used:

```bash
# Find the PID of the process running port 5432.
$ lsof -i :5432

# Kill the PID.
$ kill -9 <pid>
```
