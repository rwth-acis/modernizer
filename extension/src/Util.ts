import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import * as ini from 'ini';

interface GitConfig {
    [key: string]: {
        merge?: string;
        remote?: string;
        url?: string;
    };
}

function getGitHubRepoURL(url: string): string | null {
    if (url.endsWith('.git')) {
        url = url.substring(0, url.length - '.git'.length);
    }
    if (url.startsWith('https://github.com/') || url.startsWith('git@github.com:')) {
        return url.replace(/^git@github.com:/, 'https://github.com/');
    }
    return null;
}

async function findGitFolder(fileName: string): Promise<string> {
    let dir = path.dirname(fileName);
    const { root } = path.parse(dir);
    while (true) {
        const gitDir = path.join(dir, '.git');
        const exists = await fs.promises.access(gitDir, fs.constants.F_OK)
            .then(() => true)
            .catch(() => false);
        if (exists) {
            return gitDir;
        } else if (dir === root) {
            throw new Error('No .git dir found. Is this a git repo?');
        }
        dir = path.dirname(dir);
    }
}

async function getWorktreePath(gitPath: string): Promise<string | undefined> {
    if (fs.statSync(gitPath).isFile()) {
        const text = await fs.promises.readFile(gitPath, 'utf8');
        const worktreePrefix = 'gitdir: ';
        if (text.startsWith(worktreePrefix)) {
            return text.slice(worktreePrefix.length).trim();
        }
    }
}

export async function calculateURL(): Promise<string> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        throw new Error('No selected editor');
    }
    const { document, selection } = editor;
    const { fileName } = document;

    const gitDir = await findGitFolder(fileName);
    const baseDir = path.join(gitDir, '..');
    const worktreePath = await getWorktreePath(gitDir);
    const relativePath = path.relative(baseDir, fileName);

    const head = await fs.promises.readFile(path.join(worktreePath || gitDir, 'HEAD'), 'utf8');
    const refPrefix = 'ref: ';
    const ref = head.split('\n').find(line => line.startsWith(refPrefix));
    if (!ref) {
        throw new Error('No ref found. Cannot calculate current commit');
    }
    const refName = ref.substring(refPrefix.length);
    const sha = (await fs.promises.readFile(path.join(gitDir, refName), 'utf8')).trim();

    const gitConfig: GitConfig = ini.parse(await fs.promises.readFile(path.join(gitDir, 'config'), 'utf8'));

    const branchInfo = Object.values(gitConfig).find(val => val['merge'] === refName);
    if (!branchInfo) {
        throw new Error('No branch info found. Cannot calculate remote');
    }
    const remote = branchInfo['remote'];
    const remoteInfo = Object.entries(gitConfig).find(entry => entry[0] === `remote "${remote}"`);
    if (!remoteInfo) {
        throw new Error(`No remote found called "${remote}"`);
    }
    const url = remoteInfo[1]?.url;
    const repoURL = getGitHubRepoURL(url ?? '');
    if (!repoURL) {
        throw new Error(`The remote "${remote}" does not appear to be hosted at GitHub`);
    }

    const start = selection.start.line + 1;
    const end = selection.end.line + 1;

    const relativePathURL = relativePath.split(path.sep).join('/');
    const absolutePathURL = `${repoURL}/blob/${sha}/${relativePathURL}`;

    if (start === 1 && end === document.lineCount) {
        return absolutePathURL;
    } else if (start === end) {
        return `${absolutePathURL}#L${start}`;
    }

    return `${absolutePathURL}#L${start}-L${end}`;
}