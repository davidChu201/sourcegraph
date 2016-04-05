package localstore

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"
	"gopkg.in/gorp.v1"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/server/accesscontrol"
	"sourcegraph.com/sourcegraph/sourcegraph/server/internal/localstore/dbtypes"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
	"sourcegraph.com/sourcegraph/sourcegraph/util/dbutil"
)

func init() {
	tbl := Schema.Map.AddTableWithName(dbRegisteredClient{}, "reg_clients").SetKeys(false, "ID")
	tbl.ColMap("JWKS").SetMaxSize(5000)
	Schema.CreateSQL = append(Schema.CreateSQL,
		`ALTER TABLE reg_clients ALTER COLUMN created_at TYPE timestamp with time zone USING created_at::timestamp with time zone;`,
		`ALTER TABLE reg_clients ALTER COLUMN updated_at TYPE timestamp with time zone USING updated_at::timestamp with time zone;`,
		"ALTER TABLE reg_clients ALTER COLUMN redirect_uris TYPE text[] USING array[redirect_uris]::text[];",
		`CREATE INDEX reg_clients_authn ON reg_clients(id, client_secret_sha256);`,
	)
}

// dbRegisteredClient DB-maps a sourcegraph.RegisteredClient object.
type dbRegisteredClient struct {
	ID                 string
	ClientSecretSHA256 []byte `db:"client_secret_sha256"`
	ClientURI          string `db:"client_uri"`
	JWKS               string
	RedirectURIs       *dbutil.StringSlice `db:"redirect_uris"`
	ClientName         string              `db:"client_name"`
	Description        string
	Meta               dbtypes.JSONMapStringString
	Type               int32
	CreatedAt          *time.Time `db:"created_at"`
	UpdatedAt          *time.Time `db:"updated_at"`
}

func (u *dbRegisteredClient) toRegisteredClient() *sourcegraph.RegisteredClient {
	return &sourcegraph.RegisteredClient{
		ID: u.ID,
		// Secret field is not set because it is stored in the DB as
		// an irreversible SHA-256 hash.
		ClientURI:    u.ClientURI,
		JWKS:         u.JWKS,
		RedirectURIs: u.RedirectURIs.Slice,
		ClientName:   u.ClientName,
		Description:  u.Description,
		Meta:         map[string]string(u.Meta),
		Type:         sourcegraph.RegisteredClientType(u.Type),
		CreatedAt:    *ts(u.CreatedAt),
		UpdatedAt:    *ts(u.UpdatedAt),
	}
}

func (u *dbRegisteredClient) fromRegisteredClient(u2 *sourcegraph.RegisteredClient) {
	u.ID = u2.ID
	if u2.ClientSecret != "" {
		s := sha256.Sum256([]byte(u2.ClientSecret))
		u.ClientSecretSHA256 = s[:]
	}
	u.ClientURI = u2.ClientURI
	u.JWKS = u2.JWKS
	u.RedirectURIs = &dbutil.StringSlice{Slice: u2.RedirectURIs}
	u.ClientName = u2.ClientName
	u.Description = u2.Description
	u.Meta = u2.Meta
	u.Type = int32(u2.Type)
	u.CreatedAt = tm(&u2.CreatedAt)
	u.UpdatedAt = tm(&u2.UpdatedAt)
}

func toRegisteredClients(us []*dbRegisteredClient) []*sourcegraph.RegisteredClient {
	u2s := make([]*sourcegraph.RegisteredClient, len(us))
	for i, u := range us {
		u2s[i] = u.toRegisteredClient()
	}
	return u2s
}

// registeredClients is a DB-backed implementation of the RegisteredClients store.
type registeredClients struct{}

var _ store.RegisteredClients = (*registeredClients)(nil)

func (s *registeredClients) Get(ctx context.Context, client sourcegraph.RegisteredClientSpec) (*sourcegraph.RegisteredClient, error) {
	if err := accesscontrol.VerifyUserHasReadAccess(ctx, "RegisteredClients.Get", ""); err != nil {
		return nil, err
	}
	regClient, err := s.getBySQL(ctx, "id=$1", client.ID)
	if err, ok := err.(*store.RegisteredClientNotFoundError); ok {
		err.ID = client.ID
	}
	return regClient, err
}

func (s *registeredClients) GetByCredentials(ctx context.Context, cred sourcegraph.RegisteredClientCredentials) (*sourcegraph.RegisteredClient, error) {
	secretSHA256 := sha256.Sum256([]byte(cred.Secret))
	regClient, err := s.getBySQL(ctx, "id=$1 AND client_secret_sha256=$2", cred.ID, secretSHA256[:])
	if err, ok := err.(*store.RegisteredClientNotFoundError); ok {
		err.ID = cred.ID
		err.Secret = cred.Secret
	}
	return regClient, err
}

// getBySQL returns a client matching the SQL query (if any exists). A
// "LIMIT 1" clause is appended to the query before it is executed.
func (s *registeredClients) getBySQL(ctx context.Context, query string, args ...interface{}) (*sourcegraph.RegisteredClient, error) {
	var client dbRegisteredClient
	if err := dbh(ctx).SelectOne(&client, "SELECT * FROM reg_clients WHERE ("+query+") LIMIT 1", args...); err == sql.ErrNoRows {
		return nil, &store.RegisteredClientNotFoundError{}
	} else if err != nil {
		return nil, err
	}
	return client.toRegisteredClient(), nil
}

func (s *registeredClients) Create(ctx context.Context, client sourcegraph.RegisteredClient) error {
	if client.ID == "" {
		return fmt.Errorf("registered client ID must be set")
	}
	if client.ClientSecret == "" && client.JWKS == "" {
		return fmt.Errorf("registered client secret or JWKS must be set")
	}

	var dbClient dbRegisteredClient
	dbClient.fromRegisteredClient(&client)
	if err := dbh(ctx).Insert(&dbClient); err != nil {
		if strings.Contains(err.Error(), `duplicate key value violates unique constraint "reg_clients_pkey"`) {
			return store.ErrRegisteredClientIDExists
		}
		return err
	}
	return nil
}

func (s *registeredClients) Update(ctx context.Context, client sourcegraph.RegisteredClient) error {
	if err := accesscontrol.VerifyClientSelfOrAdmin(ctx, "RegisteredClients.Update", client.ID); err != nil {
		return err
	}
	if client.ID == "" {
		return fmt.Errorf("registered client ID must be set")
	}
	if client.ClientSecret != "" {
		return fmt.Errorf("registered client secret must not be set")
	}

	var args []interface{}
	arg := func(a interface{}) string {
		v := gorp.PostgresDialect{}.BindVar(len(args))
		args = append(args, a)
		return v
	}

	var dbClient dbRegisteredClient
	dbClient.fromRegisteredClient(&client)

	// This SQL needs to be updated whenever the fields change. It
	// can't just use the Update method because it needs to avoid
	// overwriting the Secret.
	sql := `UPDATE reg_clients SET
client_uri=` + arg(dbClient.ClientURI) + `, redirect_uris=` + arg(dbClient.RedirectURIs) + `,
client_name=` + arg(dbClient.ClientName) + `, description=` + arg(dbClient.Description) + `,
"type"=` + arg(dbClient.Type) + `, meta=` + arg(dbClient.Meta) + `
WHERE id=` + arg(dbClient.ID)

	res, err := dbh(ctx).Exec(sql, args...)
	if err != nil {
		return err
	}
	if nrows, err := res.RowsAffected(); err != nil {
		return err
	} else if nrows == 0 {
		return &store.RegisteredClientNotFoundError{ID: client.ID}
	}
	return nil
}

func (s *registeredClients) Delete(ctx context.Context, client sourcegraph.RegisteredClientSpec) error {
	if err := accesscontrol.VerifyUserHasAdminAccess(ctx, "RegisteredClients.Delete"); err != nil {
		return err
	}
	res, err := dbh(ctx).Exec(`DELETE FROM reg_clients WHERE id=$1;`, client.ID)
	if err != nil {
		return err
	}
	if nrows, err := res.RowsAffected(); err != nil {
		return err
	} else if nrows == 0 {
		return &store.RegisteredClientNotFoundError{ID: client.ID}
	}
	return nil
}

func (s *registeredClients) List(ctx context.Context, opt sourcegraph.RegisteredClientListOptions) (*sourcegraph.RegisteredClientList, error) {
	if err := accesscontrol.VerifyUserHasAdminAccess(ctx, "RegisteredClients.List"); err != nil {
		return nil, err
	}
	var args []interface{}
	arg := func(a interface{}) string {
		v := gorp.PostgresDialect{}.BindVar(len(args))
		args = append(args, a)
		return v
	}
	sql := `SELECT * FROM reg_clients WHERE true `
	if opt.Type != sourcegraph.RegisteredClientType_Any {
		sql += `AND "type" = ` + arg(opt.Type)
	}

	sql += " ORDER BY created_at DESC"

	limit := opt.PerPageOrDefault()
	sql += fmt.Sprintf(" LIMIT %s OFFSET %s", arg(limit+1), arg(opt.Offset()))

	var clients []*dbRegisteredClient
	if _, err := dbh(ctx).Select(&clients, sql, args...); err != nil {
		return nil, err
	}
	return &sourcegraph.RegisteredClientList{
		Clients: toRegisteredClients(clients),
		StreamResponse: sourcegraph.StreamResponse{
			HasMore: len(clients) == limit+1,
		},
	}, nil
}