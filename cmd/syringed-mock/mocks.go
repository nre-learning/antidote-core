package main

const (
	collectionRaw = `---
id: 1
title: Test Collection
image: https://networkreliability.engineering/images/2019/09/antidotev040.png
website: https://networkreliability.engineering
contactEmail: "blah@blah.com"
briefDescription: Terse is best.
longDescription: big long description that in no way is fake or made up. No sir.
type: vendor
tier: prod`

	lessonRaw = `---
lessonName: Antidote Test Lesson
lessonId: 1
category: fundamentals
tier: ptr
description: foobar

slug: foobar
tags:
- foo
- bar

endpoints:

- name: linux
  image: antidotelabs/utility
  presentations:
  - name: cli
    port: 22
    type: ssh

stages:
- id: 0
  description: This is a code smell you really should fix this
- id: 1
  description: Stage 1`
)
