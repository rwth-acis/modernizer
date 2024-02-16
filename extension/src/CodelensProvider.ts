import * as vscode from 'vscode';
import fetch from 'node-fetch';

export class CodelensProvider implements vscode.CodeLensProvider {
    private codeLenses: vscode.CodeLens[] = [];
    private readonly regex: RegExp;
    private _onDidChangeCodeLenses: vscode.EventEmitter<void> = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> = this._onDidChangeCodeLenses.event;

    constructor() {
        // Updated regular expression to include more top-level keywords
        this.regex = /\b(?:func|type|var|const)\b\s+([a-zA-Z_]\w*)/g;

        vscode.workspace.onDidChangeConfiguration((_) => {
            this._onDidChangeCodeLenses.fire();
        });
    }

    public async provideCodeLenses(document: vscode.TextDocument, token: vscode.CancellationToken): Promise<vscode.CodeLens[]> {
        if (vscode.workspace.getConfiguration("modernizer-vscode").get("enableCodeLens", true)) {
            this.codeLenses = [];
            const regex = new RegExp(this.regex);
            let matches;
            while ((matches = regex.exec(document.getText())) !== null) {
                const functionName = matches[1];
                const line = document.lineAt(document.positionAt(matches.index).line);
                const indexOf = line.text.indexOf(matches[0]);
                const position = new vscode.Position(line.lineNumber, indexOf);
                const range = document.getWordRangeAtPosition(position, /\b(?:func|type|var|const)\b\s+[a-zA-Z_]\w*/);
                if (range) {
                    const codeLens = new vscode.CodeLens(range);
                    
                    codeLens.command = {
                        title: `Generate Prompt for '${matches[0]}'`,
                        tooltip: `A randomized and pre-built prompt will be sent to an LLM to explain '${matches[0]}'`,
                        command: "modernizer-vscode.codelensAction",
                        arguments: [range, functionName]
                    };

                    const promptCount = await this.fetchPromptCount(functionName); // Await the fetch operation

                    const codeLens2 = new vscode.CodeLens(range, {
                        title: `Prompt Count: ${promptCount}`, // Update title with fetched prompt count
                        tooltip: `Fetching prompt count for function: ${functionName}`,
                        command: ''
                    });

                    const codeLens3 = new vscode.CodeLens(range);
                    codeLens3.command = {
                        title: "Retrieve highest ranked response",
                        command: "codelens.showInformation",
                        arguments: ["Hello from CodeLens", "Action 1", "Action 2"]
                    };

                    this.codeLenses.push(codeLens, codeLens2, codeLens3);
                }
            }
            return this.codeLenses;
        }
        return [];
    }

    public resolveCodeLens(codeLens: vscode.CodeLens, token: vscode.CancellationToken) {
        if (vscode.workspace.getConfiguration("modernizer-vscode").get("enableCodeLens", true)) {
            return codeLens;
        }
        return null;
    }

    private async fetchPromptCount(functionName: string): Promise<number | string> {
        const promptCountURL: string = 'https://modernizer.milki-psy.dbis.rwth-aachen.de/weaviate/promptcount';
        const queryParams = new URLSearchParams({ query: functionName });
        const url = `${promptCountURL}?${queryParams.toString()}`;

        const response = await fetch(url);
        if (!response.ok) {
            return "0";
        }

        const data = await response.json();
        return data;
    }
}

// Register command to show information box
let disposable = vscode.commands.registerCommand('codelens.showInformation', (message: string, action1: string, action2: string) => {
    const options: vscode.MessageItem[] = [
        { title: `👍 Upvote` },
        { title: `👎 Downvote` }
    ];
    vscode.window.showInformationMessage(message, ...options).then(selection => {
        if (selection) {
            if (selection.title.startsWith('👍')) {
                vscode.window.showInformationMessage("Upvote selected");
            } else if (selection.title.startsWith('👎')) {
                vscode.window.showInformationMessage("Downvote selected");
            }
        }
    });
});

export function activate(context: vscode.ExtensionContext) {
    // Register the CodeLens provider
    context.subscriptions.push(vscode.languages.registerCodeLensProvider('*', new CodelensProvider()));
    // Add disposables to context subscriptions
    context.subscriptions.push(disposable);
}

export function deactivate() {}
