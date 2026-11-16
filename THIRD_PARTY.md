# Native client dependencies and assets

MarkGo's Rust editor and workbench are independently implemented. The control vectors in `assets/icons/` are MarkGo's own designs; no Astrolith code or assets are included.

The pinned direct dependencies declare the following licenses:

| Dependency | Version | License |
| --- | --- | --- |
| GPUI Kit | 0.9.1 | Apache-2.0 |
| pulldown-cmark | 0.13.6 | MIT |
| unicode-segmentation | 1.13.3 | MIT OR Apache-2.0 |
| anyhow | 1.0.102 | MIT OR Apache-2.0 |
| serde | 1.0.229 | MIT OR Apache-2.0 |
| serde_json | 1.0.154 | MIT OR Apache-2.0 |
| regex | 1.13.1 | MIT OR Apache-2.0 |
| url | 2.5.8 | MIT OR Apache-2.0 |

Kit's GPUI/Base/Component code uses Apache-2.0. Its default asset provider supplies Lucide 1.43.0 icons for toolkit controls, licensed under ISC with Feather-derived icons under MIT. The complete upstream asset notice is retained in [LICENSES/LUCIDE-1.43.0.txt](LICENSES/LUCIDE-1.43.0.txt) and copied into native bundles.

`Cargo.lock` records the transitive dependency graph. This direct-dependency inventory is not a complete binary notice manifest; release packaging must assemble notices for the dependencies and assets actually distributed on each target.
