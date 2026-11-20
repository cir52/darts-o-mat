// Package site has what the admin page is about for the whole server: the settings the
// admin changes and the numbers about the server.
package site

import "github.com/uptrace/bun"

// Settings are the admin's settings of the server, one row in the database.
type Settings struct {
	bun.BaseModel `bun:"table:site_settings"`

	// ID is always 1: there is one row.
	ID int64 `bun:",pk"`
	// Registration allows anyone to create an account. A new server is open, so the first
	// account, the admin, can register.
	Registration bool `bun:",notnull,default:true"`
	// Captcha makes registering check that the visitor is a person, if the server has the
	// keys of the check; without them it does nothing. On until the admin turns it off.
	Captcha bool `bun:",notnull,default:true"`
}
