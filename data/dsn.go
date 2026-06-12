package data

import (
	"fmt"
	"strings"
)

// NewMySQLDSN builds a MySQL DSN with safe defaults.
func NewMySQLDSN(host string, port int, db, user, pass string, opts ...string) string {
	var creds string
	if user != "" {
		if pass != "" {
			creds = fmt.Sprintf("%s:%s@", user, pass)
		} else {
			creds = fmt.Sprintf("%s@", user)
		}
	}
	addr := fmt.Sprintf("tcp(%s:%d)", host, port)
	if port == 0 {
		addr = fmt.Sprintf("tcp(%s:3306)", host)
	}
	
	base := fmt.Sprintf("%s%s/%s", creds, addr, db)
	
	// Safe defaults
	params := []string{"parseTime=true", "multiStatements=true"}
	params = append(params, opts...)
	
	return base + "?" + strings.Join(params, "&")
}

// NewMariaDBDSN builds a MariaDB DSN with safe defaults.
func NewMariaDBDSN(host string, port int, db, user, pass string, opts ...string) string {
	return NewMySQLDSN(host, port, db, user, pass, opts...)
}
