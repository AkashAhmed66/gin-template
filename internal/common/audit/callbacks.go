package audit

import (
	"context"

	"gorm.io/gorm"
)

// ctxKey is the type used for stashing the request-scoped username on a context.
// security.CurrentUserKey from the security package is the public alias —
// we redeclare it here as a string constant to avoid an import cycle.
const userContextKey = "audit.current_user"

// RegisterCallbacks attaches before-create/before-update hooks that copy the
// per-request username (security.SetCurrentUsername on the context) into
// CreatedBy / UpdatedBy. Idempotent — safe to call once during boot.
func RegisterCallbacks(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").Register("audit:set_created_by", setCreatedBy); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("audit:set_updated_by", setUpdatedBy); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("audit:set_deleted_by", setDeletedBy); err != nil {
		return err
	}
	return nil
}

// SetUsernameOnContext stashes the resolved username on the context. The
// security package wraps this with a strongly-typed helper.
func SetUsernameOnContext(ctx context.Context, username string) context.Context {
	if username == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey(userContextKey), username)
}

func usernameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey(userContextKey)).(string); ok && v != "" {
		return v
	}
	return "system"
}

type ctxKey string

func setCreatedBy(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	user := usernameFromContext(db.Statement.Context)
	for _, name := range []string{"CreatedBy", "UpdatedBy"} {
		if field, ok := db.Statement.Schema.FieldsByName[name]; ok {
			db.Statement.SetColumn(field.DBName, user)
		}
	}
}

func setUpdatedBy(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	user := usernameFromContext(db.Statement.Context)
	if field, ok := db.Statement.Schema.FieldsByName["UpdatedBy"]; ok {
		db.Statement.SetColumn(field.DBName, user)
	}
}

func setDeletedBy(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	user := usernameFromContext(db.Statement.Context)
	if field, ok := db.Statement.Schema.FieldsByName["DeletedBy"]; ok {
		db.Statement.SetColumn(field.DBName, user)
	}
}
