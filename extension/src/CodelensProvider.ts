import * as vscode from 'vscode';

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

    public provideCodeLenses(document: vscode.TextDocument, token: vscode.CancellationToken): vscode.CodeLens[] | Thenable<vscode.CodeLens[]> {
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

                    const codeLens2 = new vscode.CodeLens(range);
                    codeLens2.command = {
                        title: `Retrieve similar prompts for '${matches[0]}'`,
                        tooltip: `A randomized and pre-built prompt will be sent to an LLM to retrieve similar prompts for '${matches[0]}'`,
                        command: "modernizer-vscode.codelensAction",
                        arguments: [range, functionName]
                    };

                    this.codeLenses.push(codeLens, codeLens2);
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
}
