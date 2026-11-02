module github.com/0-Mqix/packed

go 1.24.2

// v0.1.0 was released by mistake; it is v0.0.13 without its fixes. v0.1.1
// only retracts it, and itself, so that the latest version is 0.0.x again.
retract [v0.1.0, v0.1.1]

require golang.org/x/tools v0.36.0

require (
	golang.org/x/exp v0.0.0-20250819193227-8b4c13bb791b
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
)
