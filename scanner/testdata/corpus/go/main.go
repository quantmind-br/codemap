// Package main is a test corpus for validating call graph extraction.
package main

import (
	"fmt"
)

// main is the entry point - calls multiple functions.
func main() {
	greeting := hello("World")
	fmt.Println(greeting)

	result := add(1, 2)
	fmt.Printf("Result: %d\n", result)

	process()
}

// hello returns a greeting string.
func hello(name string) string {
	return "Hello, " + name
}

// add returns the sum of two integers.
func add(a, b int) int {
	return a + b
}

// process demonstrates nested calls.
func process() {
	helper()
}

// helper is called by process.
func helper() {
	nested()
}

// nested is the deepest in the call chain.
func nested() {
	fmt.Println("nested called")
}
