// Shared policy v1.0.0. Specialized adapters still validate expiry, packages and runtime.
export function projectScopedApproval(approval, expectedProject) {
  if (approval?.scope !== "project-packages-runtime") return false;
  if (
    approval.project !== expectedProject ||
    (process.env.GITHUB_REPOSITORY && process.env.GITHUB_REPOSITORY !== expectedProject) ||
    approval.releaseTag !== undefined || approval.sourceRef !== undefined
  ) throw new Error("Project-scoped approval has a mismatched project or legacy identity binding.");
  return true;
}

export function validateCandidateIdentity({ sourceSha, approvedSourceSha }) {
  if (sourceSha === undefined && approvedSourceSha === undefined) return;
  if (!/^[0-9a-f]{40}$/u.test(sourceSha ?? "") ||
      !/^[0-9a-f]{40}$/u.test(approvedSourceSha ?? "") ||
      sourceSha !== approvedSourceSha) {
    throw new Error("Candidate source SHA differs from the current workflow SHA.");
  }
}
