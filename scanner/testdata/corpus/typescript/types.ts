/**
 * Test corpus demonstrating TypeScript classes and interfaces.
 */

import { hello } from "./main";

/**
 * User entity interface.
 */
interface IUser {
  name: string;
  email: string;
  greet(): string;
}

/**
 * Represents a user entity.
 */
class User implements IUser {
  constructor(public name: string, public email: string) {}

  /**
   * Returns a greeting for the user.
   */
  greet(): string {
    return hello(this.name);
  }
}

/**
 * Handles user operations.
 */
class Service {
  private users: User[] = [];

  /**
   * Adds a user to the service.
   */
  addUser(user: User): void {
    this.users.push(user);
  }

  /**
   * Processes all users.
   */
  processAll(): void {
    for (const user of this.users) {
      user.greet();
    }
  }
}

/**
 * Factory function for creating users.
 */
function createUser(name: string, email: string): User {
  return new User(name, email);
}

export { User, Service, IUser, createUser };
