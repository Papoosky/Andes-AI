# Changelog

## [0.1.1](https://github.com/Papoosky/Andes-AI/compare/v0.1.0...v0.1.1) (2026-07-10)


### Features

* support curl-pipe install and --version flag ([7e0dbf1](https://github.com/Papoosky/Andes-AI/commit/7e0dbf15b3b978503e34c76115411afcffd8a615))


### Bug Fixes

* harden install state contracts ([2b87bcb](https://github.com/Papoosky/Andes-AI/commit/2b87bcbca83b54a02d9d623e7f4a4df9a1144bf0))
* warn on catalog ref drift ([852ee74](https://github.com/Papoosky/Andes-AI/commit/852ee74ec286628de445a5d913d3d4ade5e77c07))

## 0.1.0 (2026-07-09)


### Features

* andes doctor with drift report and nonzero exit on problems ([e6c2627](https://github.com/Papoosky/Andes-AI/commit/e6c262712ed48fe68ff0da462ebb3a5103bf881c))
* andes init non-interactive with plan display and receipt ([0d77503](https://github.com/Papoosky/Andes-AI/commit/0d775032f6effb2d385501c052c6a2cb0555f62c))
* andes list with per-skill install status ([e0ab25a](https://github.com/Papoosky/Andes-AI/commit/e0ab25a8eb0461ef0b509ad17b5b958890d368d7))
* andes update syncs mirror and refreshes skills ([ea3142b](https://github.com/Papoosky/Andes-AI/commit/ea3142b17efd87d109ac287feeb0b30e51f2747d))
* andes validate command for catalog checks ([3d674e4](https://github.com/Papoosky/Andes-AI/commit/3d674e49be34c1d45fbdcd186347595a515027ff))
* apply plan with clean skill copies ([b01eb70](https://github.com/Papoosky/Andes-AI/commit/b01eb705c9531396d3daa8662ae2ed896f0cd08e))
* banner with braille mountain logo on bare andes ([95ea131](https://github.com/Papoosky/Andes-AI/commit/95ea13169939a2171b83bf4ec4f90ef5b29b2ea8))
* bootstrap andes CLI with cobra root command ([124e8d0](https://github.com/Papoosky/Andes-AI/commit/124e8d00863b39c24caefbe2fa89030c26da82d3))
* catalog Lint for empty profiles, dup skills, frontmatter ([ae5bfe7](https://github.com/Papoosky/Andes-AI/commit/ae5bfe769947563b5107701c0e446e77df6fcc76))
* catalog loading, validation and profile resolution with fixture ([7780c8b](https://github.com/Papoosky/Andes-AI/commit/7780c8bb1cf8c6ec0d8b020baae2cfa3d8cc95f8))
* catalog resolution with git urls and baked default ([078561c](https://github.com/Papoosky/Andes-AI/commit/078561ca39e798ec8a70e04eca4040f3ed8680b4))
* derive catalog URL from repo, probing ssh then https ([4156758](https://github.com/Papoosky/Andes-AI/commit/4156758c5684dc88219418eedafa354830be3c61))
* deterministic directory content hashing ([0aaa97f](https://github.com/Papoosky/Andes-AI/commit/0aaa97fafacb7b745c0e1e8bfa5077c5b2ee711f))
* doctor diff engine over manifest, disk and catalog ([9018327](https://github.com/Papoosky/Andes-AI/commit/901832783bd7676dd8af7e2fa5c2554de61dce0c))
* end-to-end test and readme ([4e6ce93](https://github.com/Papoosky/Andes-AI/commit/4e6ce930f432d65e9e2a308d26ae2b6915d1f774))
* explain and validate catalog path prompt ([d54c412](https://github.com/Papoosky/Andes-AI/commit/d54c412ea9d1cc2a016570fb662c75055f68e123))
* git-backed catalog source with managed mirror ([27a8770](https://github.com/Papoosky/Andes-AI/commit/27a8770f8813f7544c7d38075a78e89f062e3698))
* install planning with hash-based diff ([8c8afc1](https://github.com/Papoosky/Andes-AI/commit/8c8afc1625c546ad9b1066a9a4424a8047ce8634))
* installer script and release workflow ([7e47d78](https://github.com/Papoosky/Andes-AI/commit/7e47d78a7d57cb13458d79374418668fa3fdef5e))
* interactive init with profile selection and confirm ([d7369ee](https://github.com/Papoosky/Andes-AI/commit/d7369ee8e980e57d7319c2ab4674162c11a58d73))
* interactive tui menu with in-app command output ([9ef3ffd](https://github.com/Papoosky/Andes-AI/commit/9ef3ffd2a9405854d8786d4c588cc66b4bf88de2))
* manifest catalog ref supports git url and commit sha ([ca23d1b](https://github.com/Papoosky/Andes-AI/commit/ca23d1b4cb57809b12689f143cb53cfdfe503105))
* manifest receipt with atomic save ([bdcc288](https://github.com/Papoosky/Andes-AI/commit/bdcc288a491290a77ec5d3abd4b1b0e56e509fbd))
* native in-process install flow with plan and apply screens ([83df83c](https://github.com/Papoosky/Andes-AI/commit/83df83cbec1d84022e778e38bc129d9fe0e5b8ef))
* native tui profile selection and catalog input screens ([fd1f972](https://github.com/Papoosky/Andes-AI/commit/fd1f97263352a63befe5b68297158f79ee49f449))
* probe ssh then https for the catalog url at runtime ([d9d94ba](https://github.com/Papoosky/Andes-AI/commit/d9d94ba0e0223b542eac5517aef00fc1c08c7508))
* real company logo and content-hugging tui box ([b411f0d](https://github.com/Papoosky/Andes-AI/commit/b411f0d43451fa937cd1eabd51bc452bfdee7d0b))
* show skill names in install plan and review screen ([a1f7b0c](https://github.com/Papoosky/Andes-AI/commit/a1f7b0c5bd43fb607aee731f4253611400cf5b95))
* tui freshness banner with one-key update ([665ed60](https://github.com/Papoosky/Andes-AI/commit/665ed6074c5e265760df78855fb9a985e2c1b3e5))
* unify tui aesthetic with shared bordered frame ([b5af4a8](https://github.com/Papoosky/Andes-AI/commit/b5af4a88025d84657b987e4fb000acecd175f89c))


### Bug Fixes

* center banner logo and clean stray glyphs ([7171566](https://github.com/Papoosky/Andes-AI/commit/71715662b83c012e27600dd5228180275da4c58e))
* distinguish stat errors from missing skills in catalog validation ([fd1dd90](https://github.com/Papoosky/Andes-AI/commit/fd1dd909a3f0f0504a012f72577bf49d33f44a8f))
* doctor command support for git catalogs ([643e5c6](https://github.com/Papoosky/Andes-AI/commit/643e5c6f8d0ad4b4f1d68ddff3ea2f8ae93e40c5))
* durable manifest save and spanish error wrapping ([afc43ad](https://github.com/Papoosky/Andes-AI/commit/afc43adfcc8410277467b598f36190a00e348f42))
* guard tui command dispatch against nil root factory ([679fe8e](https://github.com/Papoosky/Andes-AI/commit/679fe8eb8845a6736a9fa5946241ab792d585d8d))
* harden git invocations against argv injection ([bac9f51](https://github.com/Papoosky/Andes-AI/commit/bac9f51c767a68ec8ed87de7dd02659977d1e979))
* keep tui box width stable across screens ([50e2646](https://github.com/Papoosky/Andes-AI/commit/50e2646094bcb942eee432d8c8ff09846419403d))
* list command supports git catalogs ([7edc518](https://github.com/Papoosky/Andes-AI/commit/7edc5185db6fee86ea325161807efe4262c7d84a))
* normalize CRLF/BOM in frontmatter, pluralize counts, test on PR ([830bb9e](https://github.com/Papoosky/Andes-AI/commit/830bb9e911e326ff8ba9b45aa2ffa182333713fa))
* propagate catalog read errors in list status ([80f1588](https://github.com/Papoosky/Andes-AI/commit/80f1588949046861984f70a4a53f7364f88c8cf2))
* re-init repairs missing and modified skills at apply time ([5dca211](https://github.com/Papoosky/Andes-AI/commit/5dca211818d72320eb1b8c869ce44d9e7caec953))
* size output box to content with a max cap ([6bb706e](https://github.com/Papoosky/Andes-AI/commit/6bb706e7cd86bb812ab1e01770210bf279859b6e))
* skip confirmation when plan has no changes ([4b30112](https://github.com/Papoosky/Andes-AI/commit/4b30112a492e7e7429cd49a191928acc7d55bdb8))
* surface stat errors distinctly in doctor check ([5e59721](https://github.com/Papoosky/Andes-AI/commit/5e597210bc9faa04baeae17a834dbae770462d90))
* thread catalog path through install screen and polish input ([38d3894](https://github.com/Papoosky/Andes-AI/commit/38d38944ad3d7dae059361fdee0dd001dd211e10))
* tighten frontmatter delimiter match and cover skip-unreadable path ([141bda7](https://github.com/Papoosky/Andes-AI/commit/141bda75b390c5651b269fec96a58c1b91f7de56))
* tighten logo bbox and center it over the menu text block ([dbd072a](https://github.com/Papoosky/Andes-AI/commit/dbd072a755cf0cf8b2b060d1ec4198dcbb38d230))
* unify install apply path, thread catalog override, harden install screens ([aa0ef3d](https://github.com/Papoosky/Andes-AI/commit/aa0ef3d0f2ea5093ea676e39c0ef1a85e35a7cb2))
* update install.sh repo to Papoosky/Andes-AI ([2ae5b1c](https://github.com/Papoosky/Andes-AI/commit/2ae5b1c3b719d0a5f17b62565bd198b2a5946952))
* validate skill ids, preserve file modes, wrap doctor errors ([b64dba4](https://github.com/Papoosky/Andes-AI/commit/b64dba469e3fadfad9e3d97b7b1fa93d607e1d89))


### Continuous Integration

* release first version as 0.1.0 ([2c786dd](https://github.com/Papoosky/Andes-AI/commit/2c786dd4087f555683030864c38bafabfb037fb4))
