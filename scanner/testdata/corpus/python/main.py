"""Test corpus for validating Python call graph extraction."""

from typing import List


def main():
    """Entry point - calls multiple functions."""
    greeting = hello("World")
    print(greeting)

    result = add(1, 2)
    print(f"Result: {result}")

    process()


def hello(name: str) -> str:
    """Returns a greeting string."""
    return f"Hello, {name}"


def add(a: int, b: int) -> int:
    """Returns the sum of two integers."""
    return a + b


def process():
    """Demonstrates nested calls."""
    helper()


def helper():
    """Called by process."""
    nested()


def nested():
    """Deepest in the call chain."""
    print("nested called")


def variadic_func(*args, **kwargs):
    """A variadic function for testing arity."""
    pass


if __name__ == "__main__":
    main()
