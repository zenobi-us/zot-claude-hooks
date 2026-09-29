# Changelog

## [0.3.0](https://github.com/zenobi-us/zot-claude-hooks/compare/zot-cluade-hooks-v0.2.0...zot-cluade-hooks-v0.3.0) (2026-09-29)


### Features

* add hook management slash commands ([b4ff40c](https://github.com/zenobi-us/zot-claude-hooks/commit/b4ff40cb286334c194fb7c5e0b2d1963e17870ee))
* add safe hook environment persistence ([7bac733](https://github.com/zenobi-us/zot-claude-hooks/commit/7bac7332a58ba3eb9ca1f73930f438b546972cf7))
* align extension with Go template ([4fd2d65](https://github.com/zenobi-us/zot-claude-hooks/commit/4fd2d6573503e0bfac18abdce55f74150402a4ad))
* align hook event directory context ([34d7f15](https://github.com/zenobi-us/zot-claude-hooks/commit/34d7f158cbbfde313a94cf9b8c6dfb82dd5696e2))
* discover hooks from ZOT_HOME ([df59db6](https://github.com/zenobi-us/zot-claude-hooks/commit/df59db6c5e0cc526d96292ec797a019b842eef3e))
* expose verified zot runtime environment ([70ead27](https://github.com/zenobi-us/zot-claude-hooks/commit/70ead275a5e2b5a3206b67640b427f6e82356645))
* extract zot claude hooks extension ([bee642f](https://github.com/zenobi-us/zot-claude-hooks/commit/bee642faa90f731dd9ac9c46b42e98e85f62d426))
* forward additional zot lifecycle hooks ([0df6198](https://github.com/zenobi-us/zot-claude-hooks/commit/0df6198a5781c1191dcbd31d340c5d4917c98b7f))
* **hooks:** discover shared hooks directories (closes [#2](https://github.com/zenobi-us/zot-claude-hooks/issues/2)) ([#3](https://github.com/zenobi-us/zot-claude-hooks/issues/3)) ([c19371d](https://github.com/zenobi-us/zot-claude-hooks/commit/c19371d63d6c8f7413b1078add3fa16fc58b1875))
* **hooks:** support turn and prompt lifecycle hooks ([a83e2cc](https://github.com/zenobi-us/zot-claude-hooks/commit/a83e2cc905a2620701e0d248aac9b0fa2746dde9))
* migrate to golang ([d0640be](https://github.com/zenobi-us/zot-claude-hooks/commit/d0640be40c1c10ccceb794f9e74e6b2287c46493))
* notify when hooks activate ([532dbd5](https://github.com/zenobi-us/zot-claude-hooks/commit/532dbd50035a432529c3f78c895c336a1a8044ff))
* preserve hook environment ([da1b8c5](https://github.com/zenobi-us/zot-claude-hooks/commit/da1b8c52852d18a6084e2a4fa1e4dee5a5fce895))
* scope extension hook environment ([fcb13ae](https://github.com/zenobi-us/zot-claude-hooks/commit/fcb13aef86bf15e7638fb6bd548adfb61e07792b))
* show hook source paths in panel ([ad45d68](https://github.com/zenobi-us/zot-claude-hooks/commit/ad45d687e4ae4512430cbcdab82d9161db33780f))


### Bug Fixes

* correct Claude session alias lifecycle ([d479196](https://github.com/zenobi-us/zot-claude-hooks/commit/d47919610a7f28b83854449bb26a98a6502d17fb))
* harden hook environment persistence ([eb861de](https://github.com/zenobi-us/zot-claude-hooks/commit/eb861de92fd047fcb3feea09b9357618f56f17c9))
* **hooks:** preserve session runtime across events ([8eb4496](https://github.com/zenobi-us/zot-claude-hooks/commit/8eb44967a0e526cff8dbc5f3253f367deea0234d))
* include environment builder in extension ([a1402ce](https://github.com/zenobi-us/zot-claude-hooks/commit/a1402ce2ed67ca18b25e87184cc269ae38cc6dce))
* preserve duplicate inherited environment entries ([2f3940d](https://github.com/zenobi-us/zot-claude-hooks/commit/2f3940d3ca89e83ea09478d54fc7d9bc03a20b3d))
* resolve hook paths relative to the active project ([0300cce](https://github.com/zenobi-us/zot-claude-hooks/commit/0300cce4b1d1d7adb3c6817e2de20f6bd38f17cb))
