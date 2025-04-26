# Evo 🌿

> [!NOTE]
> Sad to realize that a good idea was buried because of cyberbullying. I will look into the current state of the project, and if this is indeed a possible **revo**lution, I will continue to support and develop it.
> 
> Why? I've always been dissatisfied with existing solutions: Git, Mercurial and others were the best in their time, but it's time to move on and create new concepts.
>
> <p align="right"><i>atarwn</i></p>

> [!IMPORTANT]
> This project has been discontinued. Due to persistent harassment and hostile behavior from certain members of the community, I have made the difficult decision to cease development of this project. While I'm proud of what was built and the vision it represented, I cannot continue maintaining it under these circumstances. The repository will remain archived for reference, but will no longer receive updates or support. Thank you to those who supported this project constructively.

> ~~**Note**: This is my hobby project in active development! While the core concepts are working, some features are still experimental and under construction. If you like the vision, contributions and feedback are very welcome! 🚧~~

Next-Generation, CRDT-Based Version Control
No Merge Conflicts • Named Streams • Stable File IDs • Large File Support

Evo 🌿 aims to evolve version control by abandoning outdated branch merges and conflict resolutions. Instead, it leverages CRDT (Conflict-Free Replicated Data Type) magic so that changes from multiple users automatically converge—no fighting with merges or losing work when files are renamed!

## Why Evo? 🌿

1. **Zero Merge Conflicts**
   The line-based RGA CRDT merges text changes from different developers seamlessly.
2. **Named Streams Instead of Branches**
   Create and switch streams for new features, merge or cherry-pick commits from one stream to another—no more complicated branching.
3. **Renames Made Simple**
   Files get stable UUIDs in .evo/index so that renames never lose history.
4. **Large File Support**
   Automatic detection moves big files to .evo/largefiles/ and stores only a stub in the CRDT logs.
5. **Offline-First**
   Commit, revert, or switch streams locally with no server required.
6. **Commit Signing**
   Optional Ed25519 signing for users who need authenticity checks.

## ~~Work in Progress~~ Project Status 🌿

~~While Evo's core is functional, there's active development on:
- Advanced partial merges for even more granular change selection
- Extended tests (unit/integration/E2E)
- Server-based PR flows for code reviews
- Performance (packfiles, caching)
- CLI & UI polish~~

This project has been discontinued and is no longer under development. The code remains as-is for reference purposes, but no further updates or improvements will be made.

~~Your feedback and contributions can help shape Evo's future!~~

## Vision 🌿

The goal is to make version control feel effortless: merges happen automatically, renames never break history, large files don't slow you down, and everything works offline. The future roadmap includes a fully realized server for pull requests, enterprise auth, and real-time collaboration—all powered by CRDT behind the scenes.

## Installing Evo 🛠️

> **Note**: As this is a hobby project, some features might not work as described. Feel free to experiment and contribute improvements!

1. Clone & Build:
```bash
git clone https://github.com/crazywolf132/evo.git
cd evo
go mod tidy
go build -o evo ./cmd/evo
```

2. (Optional) Install:
```bash
go install ./cmd/evo
```

## Quick Start 🚀

```bash
# Initialize a new Evo repo
evo init

# Check for changed or renamed files
evo status

# Commit changes (optionally sign)
evo commit -m "Initial commit"

# Create a new stream (like a branch)
evo stream create feature-x
evo stream switch feature-x

# Make changes -> evo status -> evo commit ...
# Merge everything back into main when ready
evo stream merge feature-x main
```

## Contributing 💪

This project is no longer accepting contributions as it has been discontinued. The repository is archived for reference purposes only.

## License 📜

Evo 🌿 is released under the MIT License. Hope you find it as fun and liberating to use as it is to build!

---

Thanks for checking out Evo 🌿! I'm excited to see this project grow into a conflict-free, rename-friendly, large-file-ready version control system. Remember, it's a work in progress, so expect some rough edges - but with your help, it can become amazing! ✨
