package main

import (
	"fmt"
	"os"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/doctor"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("FAIL config: %v\n", err)
		os.Exit(1)
	}
	results := doctor.RunAll(doctor.Runner{Cfg: cfg})
	exit := 0
	for _, result := range results {
		status := "PASS"
		if !result.OK {
			status = "FAIL"
			exit = 1
		}
		fmt.Printf("%s %s: %s\n", status, result.Name, result.Details)
	}
	if exit != 0 {
		os.Exit(exit)
	}
}
