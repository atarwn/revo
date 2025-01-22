# Evo Design Document

## Overview & Motivation

Evo is a next-generation version control system designed to solve problems that legacy systems (like Git) struggle with—especially around complex merges, large file handling, and rename tracking. By leveraging CRDTs (Conflict-Free Replicated Data Types), Evo can integrate changes from multiple developers without forcing manual merges or conflicts, all while supporting a familiar commit/branch-like workflow.

## Key Goals

1. **Branch-Free, Named Streams**
   - Instead of Git branches, Evo uses named streams to isolate sets of changes
   - Merging is a matter of replicating CRDT operations from one stream to another

2. **CRDT-Powered Concurrency**
   - No more "merge conflicts"
   - Evo's line-based RGA (Replicated Growable Array) CRDT automatically merges line insertions, updates, and deletions even when multiple developers modify the same file concurrently

3. **Stable File IDs for Renames**
   - Renames no longer break history
   - Evo maintains a `.evo/index` that assigns each file a stable, UUID-based ID so renaming a file doesn't lose references to its log

4. **Large File Support**
   - Files exceeding a configurable threshold are stored in `.evo/largefiles/<fileID>` with only a stub line in the CRDT logs
   - This prevents huge content from bloating the text-based logs

5. **Full Revert & Partial Merges**
   - Every commit tracks the old content on updates, allowing truly comprehensive revert
   - Partial merges (or "cherry-picks") replicate only a single commit's changes from one stream to another, as opposed to pulling everything

6. **Optional Commit Signing**
   - Evo supports Ed25519-based signatures for verifying authenticity
   - Commits store a signature field to guard against tampering

## Architecture

Below is a high-level view of Evo's architecture and rationale:

### 1. Named Streams
- Each stream is effectively a separate CRDT operation log stored in `.evo/ops/<stream>`
- Users can create or switch streams (akin to branches)
- Merging means copying missing commits (and their CRDT operations) from one stream's logs to another

**Design Decision:** This approach provides a branch-like user experience but avoids the complexity of Git merges and HEAD pointers. CRDT ensures no merge conflicts.

### 2. RGA-Based CRDT
- We employ an RGA (Replicated Growable Array) for each file, which can handle line insertion, deletion, and reordering
- The RGA logic is stored in `.evo/ops/<stream>/<fileID>.bin` in a custom binary format (no JSON overhead)
- Each operation has `(lamport, nodeID)` for concurrency ordering, plus a `lineID` for each line

**Design Decision:**
- RGA allows lines to be re-inserted anywhere, supporting reordering or partial merges with minimal overhead
- Using a binary format speeds up parsing and reduces disk usage

### 3. Stable File IDs
- `.evo/index` maps `filePath -> fileID`. If a user renames a file, we only update the index; the CRDT logs still reference the same fileID
- This ensures rename history is never lost, unlike older VCS tools that rely on heuristics to guess renames

### 4. Commits & Reverts
- A commit is a snapshot of newly added operations since the previous commit, stored in `.evo/commits/<stream>/<commitID>.bin`
- For update operations, we store the `oldContent` so revert can truly restore lines to what they were
- Revert automatically generates inverse operations (e.g., an insert becomes a delete) and re-applies them to the CRDT logs

**Design Decision:**
- By storing old content in commits, we can revert precisely, even for partial updates or line changes, avoiding the simplistic "delete everything" approach

### 5. Large File Handling
- If a file's size exceeds a configurable threshold (`files.largeThreshold`), Evo writes a CRDT stub line `EVO-LFS:<fileID>` and places the real file content into `.evo/largefiles/<fileID>/`
- This keeps the CRDT logs small and is reminiscent of Git-LFS, but simpler and built-in

### 6. Partial Merges & Cherry-Pick
- `evo stream merge <src> <target>` merges all missing commits from `<src>` to `<target>`
- `evo stream cherry-pick <commitID> <target>` merges only that single commit
- Because each commit references discrete CRDT operations by file ID, partial merges replicate exactly the needed ops

### 7. Optional Ed25519 Signing
- Users can configure a signing key path (`signing.keyPath` in config)
- On commit, Evo can create a signature by hashing the commit's stable representation (metadata + ops) and sign it
- If `verifySignatures = true`, the CLI warns if the signature fails verification

**Design Decision:**
- This approach is offline-first: no server needed
- The user's private key is local, and signatures are purely a cryptographic measure for authenticity

## CLI Summary

1. **Initialize Repository**
   ```bash
   evo init [dir]
   ```
   - Creates `.evo/` structure, "main" stream, config, etc.

2. **Configuration**
   ```bash
   evo config [get|set] ...
   ```
   - Manage global/repo-level settings (`user.name`, `user.email`, `remote origin`, etc.)

3. **Status**
   ```bash
   evo status
   ```
   - Shows changed files, new files, renames, etc.
   - Lists current stream and pending operations

4. **Commit**
   ```bash
   evo commit -m <msg> [--sign]
   ```
   - Groups newly added ops into a commit with a user-provided message, optional signing

5. **Revert**
   ```bash
   evo revert <commit-id>
   ```
   - Generates inverse ops to restore lines from a prior commit

6. **Log**
   ```bash
   evo log
   ```
   - Lists commits in the current stream, optionally verifying signatures

7. **Stream**
   ```bash
   evo stream <create|switch|list|merge|cherry-pick>
   ```
   - Manages named streams (branch-like workflows)

8. **Sync**
   ```bash
   evo sync <remote> (not fully implemented)
   ```
   - Stub for pushing/pulling CRDT logs from a future Evo server

## Config & Auth

- Global config at `~/.config/evo/config.toml`
- Repo config at `.evo/config/config.toml`
- Example keys:
  - `user.name`, `user.email`
  - `files.largeThreshold`
  - `verifySignatures` (true/false)
  - `signing.keyPath` (path to Ed25519 private key)

## Why Evo is Different

- **No Merge Conflicts:** CRDT concurrency means each line insertion, update, or deletion merges automatically
- **Renames Are Trivial:** The stable file ID approach eliminates guesswork
- **Partial Merges:** Cherry-pick or revert lines in a simpler manner thanks to the operation-based CRDT approach
- **Offline-First:** No central server required; commits and merges work locally with minimal overhead
- **Extensible:** We can add "pull requests," "server-based merges," or advanced partial file merges without rewriting the entire engine

## Conclusion

Evo aims to simplify version control while enhancing concurrency and rename support. It merges automatically using a robust line-based CRDT, organizes changes into named streams instead of ephemeral branches, and offers optional commit signing plus large file offloading.

The result is a production-ready, innovative VCS that supports both small personal projects and large enterprise codebases—offline or with a future server for collaboration. Evo's design choices reflect the vision of replacing traditional DVCS with something more powerful, more flexible, and less conflict-prone.