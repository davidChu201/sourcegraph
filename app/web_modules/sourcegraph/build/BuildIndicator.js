import React from "react";
import Component from "sourcegraph/Component";
import Dispatcher from "sourcegraph/Dispatcher";
import {urlToBuild} from "sourcegraph/build/routes";
import "sourcegraph/build/BuildBackend";
import * as BuildActions from "sourcegraph/build/BuildActions";

const endedPollInterval = 15000;
const notYetEndedPollInterval = 5000;

class BuildIndicator extends Component {
	constructor(props) {
		super(props);
		this._createBuild = this._createBuild.bind(this);
		this._interval = null;
	}

	componentWillUnmount() {
		if (this._interval !== null) clearInterval(this._interval);
	}

	// _updater periodically triggers an update of the build. It accepts a user-supplied
	// interval (msec) because we don't want to poll as frequently after we've seen a
	// successful build.
	_updater(msec) {
		if (this._interval !== null) clearInterval(this._interval);
		this._interval = setInterval(() => {
			Dispatcher.Backends.dispatch(new BuildActions.WantNewestBuildForCommit(this.state.repo, this.state.commitID, true));
		}, msec);
	}

	reconcileState(state, props) {
		state.repo = props.repo;
		state.commitID = props.commitID;
		state.branch = props.branch || null;
		state.buildable = Boolean(props.buildable);
		state.builds = props.builds;

		let builds = state.builds.listNewestByCommitID(state.repo, state.commitID);
		state.build = (builds && builds.length > 0) ? builds[0] : null;
		state.loading = builds === null;
	}

	onStateTransition(prevState, nextState) {
		if (nextState.repo !== prevState.repo || nextState.commitID !== prevState.commitID || nextState.branch !== prevState.branch) {
			Dispatcher.Backends.dispatch(new BuildActions.WantNewestBuildForCommit(nextState.repo, nextState.commitID, true));
		}
		if (nextState.build && nextState.build !== prevState.build) {
			this._updater(nextState.build.EndedAt ? endedPollInterval : notYetEndedPollInterval);
		}
	}

	_createBuild(ev) {
		Dispatcher.Backends.dispatch(new BuildActions.CreateBuild(this.state.repo, this.state.commitID, this.state.branch));
	}

	_statusLabel(b) {
		if (b.Failure) {
			return "failed";
		}
		if (b.Success) {
			return "pass";
		}
		if (b.StartedAt && !b.EndedAt) {
			return "in progress";
		}
		return "queued";
	}

	render() {
		if (this.state.loading) {
			return (
				<a key="indicator"
					className={`build-indicator btn btn-xs not-available`}>
				</a>
			);
		}

		if (this.state.build === null) {
			return (
				<a key="indicator"
					title={this.state.buildable ? "Build this version" : null}
					onClick={this.state.buildable ? this._createBuild : null}
					className={`build-indicator btn btn-xs not-available`}>
					<i className="fa fa-circle"></i>
				</a>
			);
		}

		let status = this._statusLabel(this.state.build);
		let icon, cls;
		switch (status) {
		case "failed":
			cls = "danger";
			icon = "fa-exclamation-circle";
			break;

		case "pass":
			cls = "success";
			icon = "fa-check";
			break;

		case "in progress":
			cls = "primary";
			icon = "fa-circle-o-notch fa-spin";
			break;

		case "queued":
			cls = "primary";
			icon = "fa-ellipsis-h";
			break;
		}
		return (
			<a key="indicator"
				className={`build-indicator btn btn-xs text-${cls}`}
				href={urlToBuild(this.state.repo, this.state.build.ID)}
				title={`Build #${this.state.build.ID} ${status}`}>
				<i className={`fa ${icon}`}></i>
			</a>
		);
	}
}

BuildIndicator.propTypes = {
	// repo is the repository that we are checking build data for.
	repo: React.PropTypes.string.isRequired,

	// commitID sets the revision for which we are checking build information.
	commitID: React.PropTypes.string.isRequired,

	// branch sets the branch for newly created builds. It is
	// recommended to set when creating builds, See the docs on
	// the Build.Branch field (in sourcegraph.proto) for why.
	branch: React.PropTypes.string,

	// buildable is whether or not the BuildIndicator will let the
	// user trigger a build if a build does not exist.
	buildable: React.PropTypes.bool,

	// builds is BuildStore.builds.
	builds: React.PropTypes.object.isRequired,
};

export default BuildIndicator;
