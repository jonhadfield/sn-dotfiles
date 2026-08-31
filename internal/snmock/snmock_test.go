package snmock_test

import (
	"testing"

	"github.com/jonhadfield/dotfiles-sn/internal/snmock"
	"github.com/jonhadfield/gosn-v2/auth"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/items"
	snsession "github.com/jonhadfield/gosn-v2/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signIn authenticates against the mock exactly as a real client would.
func signIn(t *testing.T, srv *snmock.Server) *snsession.Session {
	t.Helper()

	in, err := auth.CliSignIn(snmock.Email, snmock.Password, srv.URL, false)
	require.NoError(t, err)

	return &snsession.Session{
		Server:            srv.URL,
		MasterKey:         in.MasterKey,
		KeyParams:         in.KeyParams,
		AccessToken:       in.AccessToken,
		RefreshToken:      in.RefreshToken,
		AccessExpiration:  in.AccessExpiration,
		RefreshExpiration: in.RefreshExpiration,
	}
}

func newServer(t *testing.T) *snmock.Server {
	t.Helper()

	srv, err := snmock.New()
	require.NoError(t, err)
	t.Cleanup(srv.Close)

	return srv
}

func TestSignInSucceedsWithMockCredentials(t *testing.T) {
	srv := newServer(t)

	in, err := auth.CliSignIn(snmock.Email, snmock.Password, srv.URL, false)
	require.NoError(t, err)

	assert.NotEmpty(t, in.AccessToken)
	assert.NotEmpty(t, in.MasterKey)
	assert.NotZero(t, in.RefreshExpiration)
	assert.Equal(t, "004", in.KeyParams.Version)
}

func TestSignInRejectsWrongPassword(t *testing.T) {
	srv := newServer(t)

	_, err := auth.CliSignIn(snmock.Email, "not-the-password", srv.URL, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestSyncRetrievesTheSeededItemsKey(t *testing.T) {
	srv := newServer(t)
	sess := signIn(t, srv)

	so, err := items.Sync(items.SyncInput{Session: sess})
	require.NoError(t, err)

	// The items key has to arrive and decrypt, otherwise nothing else can be
	// encrypted for this account.
	require.NotEmpty(t, sess.DefaultItemsKey.ItemsKey)
	assert.Equal(t, srv.ItemsKeyUUID(), sess.DefaultItemsKey.UUID)
	assert.NotEmpty(t, so.SyncToken)
}

func TestSyncRoundTripsANote(t *testing.T) {
	srv := newServer(t)
	sess := signIn(t, srv)

	// first sync collects the items key
	_, err := items.Sync(items.SyncInput{Session: sess})
	require.NoError(t, err)

	note, err := items.NewNote("apple", "apple content", nil)
	require.NoError(t, err)

	encrypted, err := items.EncryptItem(&note, sess.DefaultItemsKey, sess)
	require.NoError(t, err)

	so, err := items.Sync(items.SyncInput{Session: sess, Items: items.EncryptedItems{encrypted}})
	require.NoError(t, err)
	require.Len(t, so.SavedItems, 1)

	// the server assigns the updated time
	assert.NotEmpty(t, so.SavedItems[0].UpdatedAt)
	assert.Len(t, srv.LiveItemsOfType(common.SNItemTypeNote), 1)

	// a fresh session must be able to read the note back and decrypt it
	reader := signIn(t, srv)

	ro, err := items.Sync(items.SyncInput{Session: reader})
	require.NoError(t, err)

	parsed, err := ro.Items.DecryptAndParse(reader)
	require.NoError(t, err)

	notes := parsed.Notes()
	require.Len(t, notes, 1)
	assert.Equal(t, "apple", notes[0].Content.GetTitle())
	assert.Equal(t, "apple content", notes[0].Content.GetText())
}

func TestSyncDeletesAnItem(t *testing.T) {
	srv := newServer(t)
	sess := signIn(t, srv)

	_, err := items.Sync(items.SyncInput{Session: sess})
	require.NoError(t, err)

	note, err := items.NewNote("lemon", "lemon content", nil)
	require.NoError(t, err)

	encrypted, err := items.EncryptItem(&note, sess.DefaultItemsKey, sess)
	require.NoError(t, err)

	_, err = items.Sync(items.SyncInput{Session: sess, Items: items.EncryptedItems{encrypted}})
	require.NoError(t, err)
	require.Len(t, srv.LiveItemsOfType(common.SNItemTypeNote), 1)

	encrypted.Deleted = true

	_, err = items.Sync(items.SyncInput{Session: sess, Items: items.EncryptedItems{encrypted}})
	require.NoError(t, err)

	assert.Empty(t, srv.LiveItemsOfType(common.SNItemTypeNote))
}

func TestSyncOnlyReturnsItemsNewToTheClient(t *testing.T) {
	srv := newServer(t)
	sess := signIn(t, srv)

	first, err := items.Sync(items.SyncInput{Session: sess})
	require.NoError(t, err)
	require.Len(t, first.Items, 1) // the items key

	// nothing has changed, so a sync with the returned token retrieves nothing
	second, err := items.Sync(items.SyncInput{Session: sess, SyncToken: first.SyncToken})
	require.NoError(t, err)
	assert.Empty(t, second.Items)
}
