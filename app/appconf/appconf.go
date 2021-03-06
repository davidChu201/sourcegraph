// Package appconf holds app configuration. The global config in this
// package is consulted by many app handlers, and if running in CLI
// mode, the config's values are set based on the CLI flags provided.
//
// This package is separate from package app to avoid import cycles
// when internal subpackages import it.
package appconf

import (
	"html/template"
	"time"

	"sourcegraph.com/sourcegraph/sourcegraph/cli/cli"
)

// Flags configure the app. The values are set by CLI flags (or during testing).
var Flags struct {
	NoUIBuild bool `long:"app.no-ui-build" description:"disable manual building of repositories from the UI" env:"SRC_APP_NO_UI_BUILD"`

	// DisableTreeEntryCommits is sometimes necessary for extremely
	// large Git repositories when `git log -- PATH` takes a very long
	// time.
	DisableTreeEntryCommits bool `long:"app.disable-tree-entry-commits" description:"do not show the latest commit for tree files/dirs" env:"SRC_APP_DISABLE_TREE_ENTRY_COMMITS"`

	DisableRepoTreeSearch bool `long:"app.disable-repo-tree-search" description:"do not show repo fulltext search results (only defs) (slower for large repos)" env:"SRC_APP_DISABLE_REPO_TREE_SEARCH"`

	DisableSearch bool `long:"app.disable-search" description:"if set, search will be entirely disabled / never allowed" env:"SRC_APP_DISABLE_SEARCH"`

	DisableApps bool `long:"app.disable-apps" description:"if set, disable the changes and issues applications" env:"SRC_APP_DISABLE_APPS"`

	DisableSupportServices bool `long:"app.disable-support-services" description:"disable 3rd party support services, including Zendesk, FullStory, Google Analytics" env:"SRC_APP_DISABLE_SUPPORT_SERVICES"`

	CustomNavLayout template.HTML `long:"app.custom-nav-layout" description:"custom layout to display in place of the search form (HTML)" env:"SRC_APP_CUSTOM_NAV_LAYOUT"`

	GoogleAnalyticsTrackingID string `long:"app.google-analytics-tracking-id" description:"Google Analytics tracking ID (UA-########-#)" env:"GOOGLE_ANALYTICS_TRACKING_ID"`

	CustomFeedbackForm template.HTML `long:"app.custom-feedback-form" description:"custom feedback form to display (HTML)" env:"CUSTOM_FEEDBACK_FORM"`

	CheckForUpdates time.Duration `long:"app.check-for-updates" description:"rate at which to check for updates and display a notification (not download/install) (0 to disable)" default:"30m" env:"SRC_APP_CHECK_FOR_UPDATES"`

	DisableExternalLinks bool `long:"app.disable-external-links" description:"Disable links to external websites" env:"SRC_APP_DISABLE_EXTERNAL_LINKS"`

	ReloadAssets bool `long:"reload" description:"(development mode only) reload app templates and other assets on each request" env:"SRC_RELOAD"`

	ExtraHeadHTML template.HTML `long:"app.extra-head-html" description:"extra HTML (<script> tags, etc.) to insert before the </head> tag" env:"SRC_APP_EXTRA_HEAD_HTML"`
	ExtraBodyHTML template.HTML `long:"app.extra-body-html" description:"extra HTML (<script> tags, etc.) to insert before the </body> tag" env:"SRC_APP_EXTRA_BODY_HTML"`

	MirrorRepoUpdateRate              time.Duration `long:"app.mirror-repo-update-rate" description:"rate at which to update mirrored repositories" default:"3s" env:"SRC_APP_MIRROR_REPO_UPDATE_RATE"`
	DisableMirrorRepoBackgroundUpdate bool          `long:"app.disable-mirror-repo-bg-update" description:"disable updating mirrored repos in the background" env:"SRC_APP_DISABLE_MIRROR_REPO_BG_UPDATE"`

	DisableGitNotify bool `long:"app.disable-git-notify" description:"disable git notifications" env:"SRC_APP_DISABLE_GIT_NOTIFY"`

	WebpackDevServerURL string `long:"app.webpack-dev-server" description:"serve frontend assets (JS, CSS, etc.) from this Webpack dev server" value-name:"URL" env:"SRC_APP_WEBPACK_DEV_SERVER"`
}

func init() {
	cli.PostInit = append(cli.PostInit, func() {
		cli.Serve.AddGroup("App", "App flags", &Flags)
	})
}
