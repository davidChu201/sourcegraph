import React from "react";

import Component from "sourcegraph/Component";
import Styles from "./styles/Footer.css";

class Footer extends Component {
	constructor(props) {
		super(props);
		const logoUrl = !global.document ? "" : document.getElementById("FooterContainer").dataset.logo;
		const versionNumber = !global.document ? "" : document.getElementById("FooterContainer").dataset.version;
		const buildVersion = !global.document ? "" : document.getElementById("FooterContainer").dataset.buildversion;
		this.state = {logo: logoUrl, version: versionNumber, build_version: buildVersion};
	}
	reconcileState(state, props) {
		Object.assign(state, props);
	}
	render() {
		return (
			<div className={Styles.footer}>
				<div className={Styles.left_container}>
					<div className={Styles.left_inline}>
						<a href="#">
							<img src={this.state.logo} />
						</a>
						<div className={Styles.address}>
							<a className={Styles.address_line} href="mailto:hi@sourcegraph.com">hi@sourcegraph.com</a>
							<span className={Styles.address_line}>121 2nd St, Ste 200</span>
							<span className={Styles.address_line}>San Francisco, CA 94105</span>
						</div>
						<span title={this.state.build_version}>
							<span className={Styles.version_number}>
								{this.state.version}
							</span>
						</span>
					</div>
				</div>
				<div className={Styles.right_container}>
					<div className={Styles.list}>
						<div className={Styles.list_header}>Company</div>
						<a className={Styles.list_element} href="/about/">About</a>
						<a className={Styles.list_element} href="/careers/">Careers</a>
						<a className={Styles.list_element} href="/blog/">Blog</a>
						<a className={Styles.list_element} href="mailto:support@sourcegraph.com">Contact</a>
					</div>
					<div className={Styles.list}>
						<div className={Styles.list_header}>Community</div>
						<a className={Styles.list_element} href="http://www.meetup.com/Sourcegraph-Hacker-Meetup/">Meetups</a>
						<a className={Styles.list_element} href="https://twitter.com/srcgraph">Twitter</a>
						<a className={Styles.list_element} href="https://www.facebook.com/sourcegraph">Facebook</a>
						<a className={Styles.list_element} href="https://www.youtube.com/channel/UCOy2N25-AHqE43XupT9mwZQ">YouTube</a>
					</div>
					<div className={Styles.list}>
						<div className={Styles.list_header}>Initiatives</div>
						<a className={Styles.list_element} href="https://srclib.org/" _target="_blank">srclib</a>
						<a className={Styles.list_element} href="https://fair.io/" _target="_blank">fair.io</a>
					</div>
					<div className={Styles.rightmost_list}>
						<div className={Styles.list_header}>Legal</div>
						<a className={Styles.list_element} href="/about/">Security</a>
						<a className={Styles.list_element} href="/privacy/">Privacy</a>
						<a className={Styles.list_element} href="/legal/">Terms</a>
					</div>
				</div>
			</div>
		);
	}
}

export default Footer;