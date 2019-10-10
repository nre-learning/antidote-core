# CHANGELOG

## In development

- Initial work on a 'mock' syringe with static sample data for integration testing [#136])https://github.com/nre-learning/syringe/pull/136)
- Added cvx and frr image names to privileged container list in config.go [#129](https://github.com/nre-learning/syringe/pull/129)
- Disable TSDB by default (configurable) [#130](https://github.com/nre-learning/syringe/pull/130)
- Cleaned up and updated deps - installing grpc tooling from source [#135](https://github.com/nre-learning/syringe/pull/135)

## v0.4.0 - August 07, 2019

- Redesigned Endpoint Abstraction (Configuration and Presentation) [#114](https://github.com/nre-learning/syringe/pull/114)
- Use the more appropriate lesson.meta.yaml instead of syringe.yaml [#101](https://github.com/nre-learning/syringe/pull/101)
- Center API and Configuration on Curriculum [#98](https://github.com/nre-learning/syringe/pull/98)
- Collections Feature and API [#104](https://github.com/nre-learning/syringe/pull/104)
- Limit volume mount to lesson directory [#109](https://github.com/nre-learning/syringe/pull/109)
- Add configuration options to influxdb export [#108](https://github.com/nre-learning/syringe/pull/108)
- Add config flag to permit egress traffic [#119](https://github.com/nre-learning/syringe/pull/119)
- Enhanced granularity for image privileges and versions [#123](https://github.com/nre-learning/syringe/pull/123)
- Fixed bug with 'allow egress' variable and added tests [#125](https://github.com/nre-learning/syringe/pull/125)
- Specify version for all platform-related docker images [#126](https://github.com/nre-learning/syringe/pull/126)
- Opened networkpolicy to all RFC1918 for the time being [#127](https://github.com/nre-learning/syringe/pull/127)
- Fix bug in jupyter version tagging [#128](https://github.com/nre-learning/syringe/pull/128)

## v0.3.2 - April 19, 2019

- Fix state bug (for real this time) and add more state tests [#100](https://github.com/nre-learning/syringe/pull/100)

## v0.3.1 - March 27, 2019

- Fixed influxdb bug [#72](https://github.com/nre-learning/syringe/pull/72)
- Add ability to use host directory for lesson content [#75](https://github.com/nre-learning/syringe/pull/75)
- Provide unit test framework for scheduler [#79](https://github.com/nre-learning/syringe/pull/79)
- Clarify difference between confusing config variables [#87](https://github.com/nre-learning/syringe/pull/87)
- Removed subnet requirement in lesson defs [#88](https://github.com/nre-learning/syringe/pull/88)
- Added validation for making sure configs are present for each stage and device [#89](https://github.com/nre-learning/syringe/pull/89)
- Add version to build info, share with syrctl [#90](https://github.com/nre-learning/syringe/pull/90)
- Setting proper permissions on copied lesson dir, using config'd location [#92](https://github.com/nre-learning/syringe/pull/92)
- Provide greater device configuration flexibility [#93](https://github.com/nre-learning/syringe/pull/93)

## v0.3.0 - February 11, 2019

- Fixed GC goroutine; make GC interval configurable [#63](https://github.com/nre-learning/syringe/pull/63)
- Add jupyter notebook as lab guide functionality [#67](https://github.com/nre-learning/syringe/pull/67)
- Added ability to perform completion verifications on livelessons [#69](https://github.com/nre-learning/syringe/pull/69)
- Re-vamp internal state systems and add external observability functions for livelessons and kubelabs [#68](https://github.com/nre-learning/syringe/pull/68)
- Changes to support Advisor functionality [#65](https://github.com/nre-learning/syringe/pull/65)

## 0.2.0 - January 24, 2019

- Simplified authentication by using consistent credentials, statically [#40](https://github.com/nre-learning/syringe/pull/40)
- Serve lab guide directly from lesson definition API [#41](https://github.com/nre-learning/syringe/pull/41)
- Simplify and improve safety of in-memory state model [#42](https://github.com/nre-learning/syringe/pull/42)
- Introduce garbage collection whitelist functionality [#45](https://github.com/nre-learning/syringe/pull/45)
- Fixed bug with bridge naming and reachability timeout [#51](https://github.com/nre-learning/syringe/pull/51)
- Add more detail around the status of a livelesson's startup progress [#52](https://github.com/nre-learning/syringe/pull/52)
- Add check to lesson import to ensure lesson IDs are unique [#53](https://github.com/nre-learning/syringe/pull/53)
- Use new githelper image instead of configmap script [#55](https://github.com/nre-learning/syringe/pull/55)
- Fix fundamentally broken networkpolicy [#58](https://github.com/nre-learning/syringe/pull/58)
- Added timeout logic to reachability test [#59](https://github.com/nre-learning/syringe/pull/59)

## 0.1.4 - January 08, 2019

- Consolidate lesson definition logic, and provide local validation tool (syrctl) [#30](https://github.com/nre-learning/syringe/pull/30)
- Redesign and fix the way iframe resources are created and presented to the API[#32](https://github.com/nre-learning/syringe/pull/32)
- Keep trying to serve metrics after an influxdb connection failure, instead of returning immediately [#35](https://github.com/nre-learning/syringe/pull/35)
- Migrate to `dep` for dependency management [#36](https://github.com/nre-learning/syringe/pull/36)
- Use the 'replace' strategy when applying config changes with NAPALM [#37](https://github.com/nre-learning/syringe/pull/37)
- Record lesson provisioning time in TSDB [#39](https://github.com/nre-learning/syringe/pull/39)

## 0.1.3 - November 15, 2018

- Modified lessondefs api to output all lessons in one call - [#18](https://github.com/nre-learning/syringe/pull/18)
- Make iframe resource image configurable - [#19](https://github.com/nre-learning/syringe/pull/19)
- Use ingresses for all iframe resources - [#28](https://github.com/nre-learning/syringe/pull/28)
- Removed some unnecessary fields from lesson metadata - [#29](https://github.com/nre-learning/syringe/pull/29)

## 0.1.1 - October 28, 2018

- Provide build info via API - [#12](https://github.com/nre-learning/syringe/pull/12), [#14](https://github.com/nre-learning/syringe/pull/14)
- Extend configurability of lessons repo to Jobs objects - [#13](https://github.com/nre-learning/syringe/pull/13)
- Deprecate "disabled" field in favor of tier - [#17](https://github.com/nre-learning/syringe/issues/17)

## v0.1.0

- Initial release, announced and made public at NXTWORK 2018 in Las Vegas
