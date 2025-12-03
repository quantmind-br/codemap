// Package main demonstrates method calls and type definitions.
package main

// User represents a user entity.
type User struct {
	Name  string
	Email string
}

// NewUser creates a new User instance.
func NewUser(name, email string) *User {
	return &User{Name: name, Email: email}
}

// Greet returns a greeting for the user.
func (u *User) Greet() string {
	return hello(u.Name)
}

// Service handles user operations.
type Service struct {
	users []*User
}

// AddUser adds a user to the service.
func (s *Service) AddUser(u *User) {
	s.users = append(s.users, u)
}

// ProcessAll processes all users - calls variadic function.
func (s *Service) ProcessAll(opts ...string) {
	for _, u := range s.users {
		_ = u.Greet()
	}
}
