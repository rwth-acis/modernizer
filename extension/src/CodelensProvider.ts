import * as vscode from 'vscode';
import fetch from 'node-fetch';
import { getSelectedFunctionRange } from './extension';

export class CodelensProvider implements vscode.CodeLensProvider {
    private codeLenses: vscode.CodeLens[] = [];
    private readonly regex: RegExp;
    private readonly _onDidChangeCodeLenses: vscode.EventEmitter<void> = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> = this._onDidChangeCodeLenses.event;

    constructor() {
        this.regex = /\b(?:class|interface|function|method|struct|enum|protocol|package|namespace|module|let|const|var|func|type|var|const)\b\s+([a-zA-Z_]\w*)/g;

        vscode.workspace.onDidChangeConfiguration(() => {
            this._onDidChangeCodeLenses.fire();
        });
    }

    public async provideCodeLenses(document: vscode.TextDocument, token: vscode.CancellationToken): Promise<vscode.CodeLens[]> {
        if (!vscode.workspace.getConfiguration("modernizer-vscode").get("enableCodeLens", true)) {
            return [];
        }

        this.codeLenses = [];
        const regex = new RegExp(this.regex);
        let matches;

        while ((matches = regex.exec(document.getText())) !== null) {
            const functionName = matches[1];
            const line = document.lineAt(document.positionAt(matches.index).line);
            const indexOf = line.text.indexOf(matches[0]);
            const position = new vscode.Position(line.lineNumber, indexOf);
            const range = document.getWordRangeAtPosition(position, /\b(?:class|interface|function|method|struct|enum|protocol|package|namespace|module|let|const|var|func|type|var|const)\b\s+([a-zA-Z_]\w*)/);

            if (range) {
                const codeLens = new vscode.CodeLens(range);
                
                codeLens.command = {
                    title: `Generate Prompt for '${matches[0]}'`,
                    tooltip: `A randomized and pre-built prompt will be sent to an LLM to explain '${matches[0]}'`,
                    command: "modernizer-vscode.codelensAction",
                    arguments: [range, functionName]
                };

                const promptCount = await fetchPromptCount(functionName);

                const codeLens2 = new vscode.CodeLens(range, {
                    title: `Prompt Count: ${promptCount}`,
                    tooltip: `Fetching prompt count for function: ${functionName}`,
                    command: ''
                });

                const codeLens3 = new vscode.CodeLens(range);
                codeLens3.command = {
                    title: "Retrieve highest ranked response",
                    command: "codelens.showInformation",
                    arguments: ["How did you like this Response?", "Action 1", "Action 2"]
                };

                this.codeLenses.push(codeLens, codeLens2, codeLens3);
            }
        }

        return this.codeLenses;
    }

    public resolveCodeLens(codeLens: vscode.CodeLens, token: vscode.CancellationToken) {
        if (!vscode.workspace.getConfiguration("modernizer-vscode").get("enableCodeLens", true)) {
            return null;
        }

        return codeLens;
    }
}

async function fetchPromptCount(functionName: string): Promise<number | string> {

    const promptCountURL: string = 'http://192.168.10.163:8080/weaviate/promptcount';
    const queryParams = new URLSearchParams({ query: functionName });
    const url = `${promptCountURL}?${queryParams.toString()}`;

    const response = await fetch(url);
    if (!response.ok) {
        return "0";
    }

    const data = await response.json();
    return data;
}

// Register command to show information box
let disposable = vscode.commands.registerCommand('codelens.showInformation', async () => {
    try {

        const activeEditor = vscode.window.activeTextEditor;
        if (activeEditor) {
            const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
            const functionCode = activeEditor.document.getText(selectedFunctionRange);

            // Fetch response
            const responseText = await fetchResponse(functionCode);

            // Create and show output window
            const outputWindow = vscode.window.createOutputChannel('Ollama Response');
            outputWindow.show(true);
            outputWindow.append(responseText.toString());
        }

        // Show voting options
        const options: vscode.MessageItem[] = [
            { title: `👍 Upvote` },
            { title: `👎 Downvote` }
        ];
        const selection = await vscode.window.showInformationMessage('Vote for this prompt:', ...options);
        if (selection) {
            if (selection.title.startsWith('👍')) {
                try {
                    //Vote(promptId, true);
                    vscode.window.showInformationMessage("Upvote selected");
                } catch (error: any) { // Explicitly type 'error' as 'any'

                    vscode.window.showErrorMessage("Failed to upvote: " + error.message);
                }
            } else if (selection.title.startsWith('👎')) {
                vscode.window.showInformationMessage("Downvote selected");
            }
        }
    } catch (error: any) {
        vscode.window.showErrorMessage("Error: " + error.message);
    }
});

async function fetchResponse(functionName: string): Promise<number | string> {
    const retrieveResponseURL: string = 'http://192.168.10.163:8080/weaviate/retrieveresponse';
    const queryParams = new URLSearchParams({ query: functionName });
    const url = `${retrieveResponseURL}?${queryParams.toString()}`;

    const response = await fetch(url);
    if (!response.ok) {
        return "0";
    }

    const data = await response.json();
    return data;
}


export function activate(context: vscode.ExtensionContext) {
    // Register the CodeLens provider
    context.subscriptions.push(vscode.languages.registerCodeLensProvider('*', new CodelensProvider()));
    // Add disposables to context subscriptions
    context.subscriptions.push(disposable);
}

export function deactivate() {}
