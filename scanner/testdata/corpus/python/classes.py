"""Test corpus demonstrating Python classes and method calls."""

from typing import List, Optional


class User:
    """Represents a user entity."""

    def __init__(self, name: str, email: str):
        """Initialize user with name and email."""
        self.name = name
        self.email = email

    def greet(self) -> str:
        """Returns a greeting for the user."""
        from main import hello
        return hello(self.name)


class Service:
    """Handles user operations."""

    def __init__(self):
        """Initialize service with empty user list."""
        self.users: List[User] = []

    def add_user(self, user: User) -> None:
        """Adds a user to the service."""
        self.users.append(user)

    def process_all(self) -> None:
        """Processes all users."""
        for user in self.users:
            _ = user.greet()

    def _private_method(self) -> None:
        """Private method - should not be exported."""
        pass


def create_user(name: str, email: str) -> User:
    """Factory function for creating users."""
    return User(name, email)
