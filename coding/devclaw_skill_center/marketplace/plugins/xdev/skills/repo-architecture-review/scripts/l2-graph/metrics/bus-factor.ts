export interface Ownership {
  file: string;
  author: string;
  addedLines: number;
}

export interface BusFactorResult {
  truckFactor: number;
  authorsRemoved: string[];
  orphansIfRemoved: string[];
}

export function computeBusFactor(ownerships: Ownership[]): BusFactorResult {
  // Group by file and find majority author (>50% added lines)
  const fileToOwner = new Map<string, string | null>();

  // Get unique files
  const files = new Set(ownerships.map((o) => o.file));

  for (const file of files) {
    const fileOwnerships = ownerships.filter((o) => o.file === file);
    const totalLines = fileOwnerships.reduce((sum, o) => sum + o.addedLines, 0);

    // Find author with > 50%
    let owner: string | null = null;
    const authorLines = new Map<string, number>();
    for (const { author, addedLines } of fileOwnerships) {
      authorLines.set(author, (authorLines.get(author) ?? 0) + addedLines);
    }

    for (const [author, lines] of authorLines) {
      if (lines > totalLines / 2) {
        owner = author;
        break;
      }
    }

    fileToOwner.set(file, owner);
  }

  // Greedy removal: remove top authors until >= 50% of files have no owner
  const allFiles = Array.from(files);
  const totalFiles = allFiles.length;
  const targetUnowned = Math.ceil(totalFiles / 2);

  // Count files owned by each author
  const authorToFiles = new Map<string, Set<string>>();
  for (const [file, owner] of fileToOwner) {
    if (owner) {
      if (!authorToFiles.has(owner)) {
        authorToFiles.set(owner, new Set());
      }
      authorToFiles.get(owner)!.add(file);
    }
  }

  // Sort authors by number of owned files (descending)
  const sortedAuthors = Array.from(authorToFiles.entries())
    .sort((a, b) => b[1].size - a[1].size)
    .map(([author]) => author);

  const authorsRemoved: string[] = [];
  const activeOwners = new Set<string>();
  for (const [, owner] of fileToOwner) {
    if (owner) activeOwners.add(owner);
  }

  for (const author of sortedAuthors) {
    authorsRemoved.push(author);
    activeOwners.delete(author);

    // Count unowned files
    let unownedCount = 0;
    for (const file of allFiles) {
      const owner = fileToOwner.get(file);
      if (!owner || !activeOwners.has(owner)) {
        unownedCount++;
      }
    }

    if (unownedCount >= targetUnowned) {
      break;
    }
  }

  // Compute orphans
  const orphans: string[] = [];
  for (const file of allFiles) {
    const owner = fileToOwner.get(file);
    if (!owner || !activeOwners.has(owner)) {
      orphans.push(file);
    }
  }

  return {
    truckFactor: authorsRemoved.length,
    authorsRemoved,
    orphansIfRemoved: orphans,
  };
}
