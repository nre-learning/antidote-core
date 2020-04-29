# CHANGELOG

## In development

- Fix livelesson leak / livesession bug [#171](https://github.com/nre-learning/antidote-core/pull/171)
- Allow travis to install binaries from vendored libs too [#172](https://github.com/nre-learning/antidote-core/pull/172)

## v0.6.0 - April 18, 2020

- Top-to-bottom revamp and move to Antidote-Core [#141](https://github.com/nre-learning/antidote-core/pull/141)
- Make travis happy [#155](https://github.com/nre-learning/antidote-core/pull/155)
- Add protections to antidote CLI to protect against landmines [#156](https://github.com/nre-learning/antidote-core/pull/156)
- Minor lesson model updates [#157](https://github.com/nre-learning/antidote-core/pull/157)
- Add mutex to podready function [#158](https://github.com/nre-learning/antidote-core/pull/158)
- Make k8s configuration configurable at Antidote level [#159](https://github.com/nre-learning/antidote-core/pull/159)
- Changed resource order on ingest, and added early return on error [#160](https://github.com/nre-learning/antidote-core/pull/160)
- Add "Authors" to lesson metadata [#162](https://github.com/nre-learning/antidote-core/pull/162)
- Ingest curriculum info, offer via API [#163](https://github.com/nre-learning/antidote-core/pull/163)
- Add functionality to attach a pullsecret to pods (allows private image pulls) [#164](https://github.com/nre-learning/antidote-core/pull/164)
- Add links to object ref documentation wherever possible in creation wizard [#165](https://github.com/nre-learning/antidote-core/pull/165)
- Minor modifications to facilitate antidote-web development  [#166](https://github.com/nre-learning/antidote-core/pull/166)
- Strip X-Frame-Options header [#167](https://github.com/nre-learning/antidote-core/pull/167)

## v0.5.1 - February 17, 2020

- Move away from runtime git clone model [#153](https://github.com/nre-learning/antidote-core/pull/153)

## v0.5.0 - February 01, 2020

- Initial work on a 'mock' syringe with static sample data for integration testing [#136])https://github.com/nre-learning/antidote-core/pull/136)
- Added cvx and frr image names to privileged container list in config.go [#129](https://github.com/nre-learning/antidote-core/pull/129)
- Disable TSDB by default (configurable) [#130](https://github.com/nre-learning/antidote-core/pull/130)
- Cleaned up and updated deps - installing grpc tooling from source [#135](https://github.com/nre-learning/antidote-core/pull/135)
- Change SYRINGE_DOMAIN to optional variable, and provide default [#142](https://github.com/nre-learning/antidote-core/pull/142)
- Add config option to control imagepullpolicy [#145](https://github.com/nre-learning/antidote-core/pull/145)
- De-couple namespaces from tier - implement syringe ID option [#150](https://github.com/nre-learning/antidote-core/pull/150)
- Move to DNS-based identifiers for http presentations [#151](https://github.com/nre-learning/antidote-core/pull/151)

## v0.4.0 - August 07, 2019

- Redesigned Endpoint Abstraction (Configuration and Presentation) [#114](https://github.com/nre-learning/antidote-core/pull/114)
- Use the more appropriate lesson.meta.yaml instead of syringe.yaml [#101](https://github.com/nre-learning/antidote-core/pull/101)
- Center API and Configuration on Curriculum [#98](https://github.com/nre-learning/antidote-core/pull/98)
- Collections Feature and API [#104](https://github.com/nre-learning/antidote-core/pull/104)
- Limit volume mount to lesson directory [#109](https://github.com/nre-learning/antidote-core/pull/109)
- Add configuration options to influxdb export [#108](https://github.com/nre-learning/antidote-core/pull/108)
- Add config flag to permit egress traffic [#119](https://github.com/nre-learning/antidote-core/pull/119)
- Enhanced granularity for image privileges and versions [#123](https://github.com/nre-learning/antidote-core/pull/123)
- Fixed bug with 'allow egress' variable and added tests [#125](https://github.com/nre-learning/antidote-core/pull/125)
- Specify version for all platform-related docker images [#126](https://github.com/nre-learning/antidote-core/pull/126)
- Opened networkpolicy to all RFC1918 for the time being [#127](https://github.com/nre-learning/antidote-core/pull/127)
- Fix bug in jupyter version tagging [#128](https://github.com/nre-learning/antidote-core/pull/128)

## v0.3.2 - April 19, 2019

- Fix state bug (for real this time) and add more state tests [#100](https://github.com/nre-learning/antidote-core/pull/100)

## v0.3.1 - March 27, 2019

- Fixed influxdb bug [#72](https://github.com/nre-learning/antidote-core/pull/72)
- Add ability to use host directory for lesson content [#75](https://github.com/nre-learning/antidote-core/pull/75)
- Provide unit test framework for scheduler [#79](https://github.com/nre-learning/antidote-core/pull/79)
- Clarify difference between confusing config variables [#87](https://github.com/nre-learning/antidote-core/pull/87)
- Removed subnet requirement in lesson defs [#88](https://github.com/nre-learning/antidote-core/pull/88)
- Added validation for making sure configs are present for each stage and device [#89](https://github.com/nre-learning/antidote-core/pull/89)
- Add version to build info, share with syrctl [#90](https://github.com/nre-learning/antidote-core/pull/90)
- Setting proper permissions on copied lesson dir, using config'd location [#92](https://github.com/nre-learning/antidote-core/pull/92)
- Provide greater device configuration flexibility [#93](https://github.com/nre-learning/antidote-core/pull/93)

## v0.3.0 - February 11, 2019

- Fixed GC goroutine; make GC interval configurable [#63](https://github.com/nre-learning/antidote-core/pull/63)
- Add jupyter notebook as lab guide functionality [#67](https://github.com/nre-learning/antidote-core/pull/67)
- Added ability to perform completion verifications on livelessons [#69](https://github.com/nre-learning/antidote-core/pull/69)
- Re-vamp internal state systems and add external observability functions for livelessons and kubelabs [#68](https://github.com/nre-learning/antidote-core/pull/68)
- Changes to support Advisor functionality [#65](https://github.com/nre-learning/antidote-core/pull/65)

## 0.2.0 - January 24, 2019

- Simplified authentication by using consistent credentials, statically [#40](https://github.com/nre-learning/antidote-core/pull/40)
- Serve lab guide directly from lesson definition API [#41](https://github.com/nre-learning/antidote-core/pull/41)
- Simplify and improve safety of in-memory state model [#42](https://github.com/nre-learning/antidote-core/pull/42)
- Introduce garbage collection whitelist functionality [#45](https://github.com/nre-learning/antidote-core/pull/45)
- Fixed bug with bridge naming and reachability timeout [#51](https://github.com/nre-learning/antidote-core/pull/51)
- Add more detail around the status of a livelesson's startup progress [#52](https://github.com/nre-learning/antidote-core/pull/52)
- Add check to lesson import to ensure lesson IDs are unique [#53](https://github.com/nre-learning/antidote-core/pull/53)
- Use new githelper image instead of configmap script [#55](https://github.com/nre-learning/antidote-core/pull/55)
- Fix fundamentally broken networkpolicy [#58](https://github.com/nre-learning/antidote-core/pull/58)
- Added timeout logic to reachability test [#59](https://github.com/nre-learning/antidote-core/pull/59)

## 0.1.4 - January 08, 2019

- Consolidate lesson definition logic, and provide local validation tool (syrctl) [#30](https://github.com/nre-learning/antidote-core/pull/30)
- Redesign and fix the way iframe resources are created and presented to the API[#32](https://github.com/nre-learning/antidote-core/pull/32)
- Keep trying to serve metrics after an influxdb connection failure, instead of returning immediately [#35](https://github.com/nre-learning/antidote-core/pull/35)
- Migrate to `dep` for dependency management [#36](https://github.com/nre-learning/antidote-core/pull/36)
- Use the 'replace' strategy when applying config changes with NAPALM [#37](https://github.com/nre-learning/antidote-core/pull/37)
- Record lesson provisioning time in TSDB [#39](https://github.com/nre-learning/antidote-core/pull/39)

## 0.1.3 - November 15, 2018

- Modified lessondefs api to output all lessons in one call - [#18](https://github.com/nre-learning/antidote-core/pull/18)
- Make iframe resource image configurable - [#19](https://github.com/nre-learning/antidote-core/pull/19)
- Use ingresses for all iframe resources - [#28](https://github.com/nre-learning/antidote-core/pull/28)
- Removed some unnecessary fields from lesson metadata - [#29](https://github.com/nre-learning/antidote-core/pull/29)

## 0.1.1 - October 28, 2018

- Provide build info via API - [#12](https://github.com/nre-learning/antidote-core/pull/12), [#14](https://github.com/nre-learning/antidote-core/pull/14)
- Extend configurability of lessons repo to Jobs objects - [#13](https://github.com/nre-learning/antidote-core/pull/13)
- Deprecate "disabled" field in favor of tier - [#17](https://github.com/nre-learning/antidote-core/issues/17)

## v0.1.0

- Initial release, announced and made public at NXTWORK 2018 in Las Vegas
