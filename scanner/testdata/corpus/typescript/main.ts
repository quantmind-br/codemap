/**
 * Test corpus for validating TypeScript call graph extraction.
 */

/**
 * Entry point - calls multiple functions.
 */
function main(): void {
  const greeting = hello("World");
  console.log(greeting);

  const result = add(1, 2);
  console.log(`Result: ${result}`);

  process();
}

/**
 * Returns a greeting string.
 */
function hello(name: string): string {
  return `Hello, ${name}`;
}

/**
 * Returns the sum of two integers.
 */
function add(a: number, b: number): number {
  return a + b;
}

/**
 * Demonstrates nested calls.
 */
function process(): void {
  helper();
}

/**
 * Called by process.
 */
function helper(): void {
  nested();
}

/**
 * Deepest in the call chain.
 */
function nested(): void {
  console.log("nested called");
}

/**
 * A variadic function for testing arity.
 */
function variadicFunc(...args: string[]): void {
  // No-op
}

main();
