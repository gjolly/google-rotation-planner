package localcred

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/gjolly/google-rotation-planner/internal/auth"
	"google.golang.org/api/calendar/v3"
)

const (
	// Keep in sync with https://rclone.org/drive/#making-your-own-client-id.
	// They hit the exact same issues with Google Drive.
	credentialsMissingMsg = `The credentials are not initialized.

To do so, head to https://console.developers.google.com

1. Create a new project if you don't have one.
1. Go to 'Enable API and services', search for Calendar API and enable it.
2. Go to 'OAuth consent screen'.
    2a. If your account is managed by an organization, you have to
        select 'Internal' as 'User Type'. For individual accounts
        select 'External'.
    2b. Set an application name (e.g. 'google-rotation-planner').
    2c. Use your email for 'User support email' and 'Developer
        contact information'. Save and continue.
    3c. Select 'Add or remove scopes' and add:
		* https://www.googleapis.com/auth/calendar.events
    3d. Save and continue until you're back to the dashboard.
3. You now have a choice. You can either:
    * Click on 'Publish App' and avoid 'Submitting for
      verification'. This will result in scary confirmation
      screens or error messages when you authorize gmailctl with
      your account (but for some users it works), OR
    * You could add your email as 'Test user' and keep the app in
      'Testing' mode. In this case everything will work, but
      you'll have to login and confirm the access every week (token
      expiration).
4.  Go to Credentials on the left.
    4a. Click 'Create credentials'.
    4b. Select 'OAuth client ID'.
    4c. Select 'Desktop app' as 'Application type' and give it a name.
    4d. Create.
5. Download the credentials file into %q and execute the 'init'
   command again.

Documentation about Google Calendar API authorization can be found
at: https://developers.google.com/identity/protocols/oauth2/scopes#calendar
`
	authMessage = `Go to the following link in your browser and authorize google-rotation-planner:

%v

NOTE that google-rotation-planner runs a webserver on your local machine to
collect the token as returned from Google. This only runs until the token is
saved. If your browser is on another machine without access to the local
network, this will not work.
`
)

// Provider is a Google Calendar credential provider that uses the local filesystem.
type Provider struct{}

func (Provider) Service(ctx context.Context, cfgDir string) (*calendar.Service, error) {
	auth, err := openCredentials(credentialsPath(cfgDir))
	if err != nil {
		return nil, err
	}
	return openToken(ctx, auth, tokenPath(cfgDir))
}

func (Provider) InitConfig(cfgDir string) error {
	cpath := credentialsPath(cfgDir)
	tpath := tokenPath(cfgDir)

	auth, err := openCredentials(cpath)
	if err != nil {
		fmt.Printf(credentialsMissingMsg, cpath)
		return err
	}
	_, err = openToken(context.Background(), auth, tpath)
	if err != nil {
		stderrPrintf("%v\n\n", err)
		err = setupToken(auth, tpath)
	}
	return err
}

func (Provider) ResetConfig(cfgDir string) error {
	if err := deleteFile(credentialsPath(cfgDir)); err != nil {
		return err
	}
	if err := deleteFile(tokenPath(cfgDir)); err != nil {
		return err
	}
	return nil
}

func (Provider) RefreshToken(ctx context.Context, cfgDir string) error {
	auth, err := openCredentials(credentialsPath(cfgDir))
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	svc, err := openToken(ctx, auth, tokenPath(cfgDir))
	if err != nil {
		return setupToken(auth, tokenPath(cfgDir))
	}
	// Check whether the token works by getting the calendars.
	if _, err := svc.CalendarList.List().Context(ctx).Do(); err != nil {
		return setupToken(auth, tokenPath(cfgDir))
	}
	return nil
}

func openCredentials(path string) (*auth.Authenticator, error) {
	cred, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening credentials: %w", err)
	}
	return auth.NewAuthenticator(cred)
}

func openToken(ctx context.Context, auth *auth.Authenticator, path string) (*calendar.Service, error) {
	token, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("missing or invalid cached token: %w", err)
	}
	return auth.Service(ctx, token)
}

func setupToken(auth *auth.Authenticator, path string) error {
	localSrv := newOauth2Server(auth.State)
	addr, err := localSrv.Start()
	if err != nil {
		return fmt.Errorf("starting local server: %w", err)
	}
	defer localSrv.Close()

	fmt.Printf(authMessage, auth.AuthURL("http://"+addr))
	authCode := localSrv.WaitForCode()
	if err := saveToken(path, authCode, auth); err != nil {
		return fmt.Errorf("caching token: %w", err)
	}
	return nil
}

func saveToken(path, authCode string, auth *auth.Authenticator) (err error) {
	fmt.Printf("Saving credential file to %s\n", path)
	f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if e != nil {
		return fmt.Errorf("creating token file: %w", e)
	}
	defer func() {
		e = f.Close()
		// Do not hide more important errors.
		if err == nil {
			err = e
		}
	}()

	return auth.CacheToken(context.Background(), authCode, f)
}

func credentialsPath(cfgDir string) string {
	return path.Join(cfgDir, "credentials.json")
}

func tokenPath(cfgDir string) string {
	return path.Join(cfgDir, "token.json")
}

func deleteFile(path string) error {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func stderrPrintf(format string, a ...interface{}) {
	/* #nosec */
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
