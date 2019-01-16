# CHANGELOG

## In development

- Simplified authentication by using consistent credentials, statically [#40](https://github.com/nre-learning/syringe/pull/40)
- Serve lab guide directly from lesson definition API [#41](https://github.com/nre-learning/syringe/pull/41)
- Simplify and improve safety of in-memory state model [#42](https://github.com/nre-learning/syringe/pull/42)
- Introduce garbage collection whitelist functionality [#45](https://github.com/nre-learning/syringe/pull/45)
- Fixed bug with bridge naming and reachability timeout [#51](https://github.com/nre-learning/syringe/pull/51)
- Add more detail around the status of a livelesson's startup progress [#52](https://github.com/nre-learning/syringe/pull/52)
- Add check to lesson import to ensure lesson IDs are unique [#53](https://github.com/nre-learning/syringe/pull/53)
- Added metadata to lesson to highlight Juniper-specific content [#54](https://github.com/nre-learning/syringe/pull/54)

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
