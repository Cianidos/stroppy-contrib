package main

import (
	workload "example.com/selectfile/workload"
	stroppy "github.com/stroppy-io/stroppy/v6"
)

func main() { stroppy.Main(workload.Test) }
