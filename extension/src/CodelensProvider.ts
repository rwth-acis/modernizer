import * as vscode from 'vscode';
import fetch from 'node-fetch';
import { getSelectedFunctionRange } from './extension';
import { Vote } from './VotingMechanism';

export let remainingResponseList: string[] = [];

export class CodelensProvider implements vscode.CodeLensProvider {

    private codeLenses: vscode.CodeLens[] = [];
    private readonly regex: RegExp;
    private readonly _onDidChangeCodeLenses: vscode.EventEmitter<void> = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> = this._onDidChangeCodeLenses.event;

    
    constructor() {
        this.regex = /\b(?:func|type|var|def|public\s+class|class|function)\b\s+([a-zA-Z_]\w*)|\bfunc\s*\(\s*\*\s*[a-zA-Z_]\w*\s*\)\s*([a-zA-Z_]\w*)|function\s+([a-zA-Z_]\w*)/g

        vscode.workspace.onDidChangeConfiguration(() => {
            this._onDidChangeCodeLenses.fire();
        });
    }

    public async provideCodeLenses(document: vscode.TextDocument, token: vscode.CancellationToken): Promise<vscode.CodeLens[]> {

        if (document.uri.scheme !== 'file') {
            if (document.uri.scheme === 'output') {
                const codeLenses1 = await this.createOutputWindowCodeLenses(document);
                const codeLenses2 = await this.createOutputWindowCodeLensesGenerate(document);
                return [...codeLenses1, ...codeLenses2];
            } else {
                return [];
            }
        }
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
            const range = document.getWordRangeAtPosition(position, this.regex);

            if (range) {
                const codeLens = new vscode.CodeLens(range);
                
                codeLens.command = {
                    title: `Explain the Function '${matches[0]}'`,
                    tooltip: `A randomized and pre-built prompt will be sent to an LLM to explain '${matches[0]}'`,
                    command: "modernizer-vscode.randomExplanationPrompt",
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
                    title: "Retrieve best response",
                    command: "modernizer-vscode.showBestResponse",
                    arguments: ["How did you like this Response?", "Action 1", "Action 2"]
                };

                this.codeLenses.push(codeLens, codeLens2, codeLens3);
            }
        }

        return this.codeLenses;
    }

    private async createOutputWindowCodeLenses(document: vscode.TextDocument): Promise<vscode.CodeLens[]> {
        const codeLenses: vscode.CodeLens[] = [];
    
        const text = document.getText();
        const regex = /The instruct used for this prompt: /g;
        let match;
        while ((match = regex.exec(text))) {
            const line = document.lineAt(document.positionAt(match.index).line);
            const position = new vscode.Position(line.lineNumber, match.index);
            const range = new vscode.Range(position, position.with(undefined, match.index + match[0].length));
        
            const gitURL = await getGitURLByID(remainingResponseList[0]);
        
            const codeLens2 = new vscode.CodeLens(range, {
                title: "Open GitHub Repository",
                command: "modernizer-vscode.openGitHubRepo",
                arguments: [gitURL]
            });
        
            codeLenses.push(codeLens2);
        
            if (remainingResponseList.length > 1) {
                const codeLens = new vscode.CodeLens(range, {
                    title: "Show next Response ⮞",
                    tooltip: "Show next Response",
                    command: "modernizer-vscode.showNextResponse"
                });
            
                codeLenses.push(codeLens);
            }
        }
    
        return codeLenses;
    }

    private async createOutputWindowCodeLensesGenerate(document: vscode.TextDocument): Promise<vscode.CodeLens[]> {
        const codeLenses: vscode.CodeLens[] = [];
    
        const text = document.getText();
        const regex = /Generated new response with the instruct: /g;
        let match;
        while ((match = regex.exec(text))) {
            const line = document.lineAt(document.positionAt(match.index).line);
            const position = new vscode.Position(line.lineNumber, match.index);
            const range = new vscode.Range(position, position.with(undefined, match.index + match[0].length));
        
        
            const codeLens = new vscode.CodeLens(range, {
                title: "Save custom Prompt to local settings.json",
                command: "modernizer-vscode.savePrompt",
            });
        
            codeLenses.push(codeLens);
        }
    
        return codeLenses;
    }
}

async function getGitURLByID(id: string): Promise<string> {
    const properties = await GetPropertiesByID(id);
    return properties.gitURL;
}

async function getResponseList(code: string): Promise<string[]> {
    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const responseListPath: string = '/weaviate/retrieveresponselist';
    const url: string = `${baseUrl}${responseListPath}`;

    const queryParams = new URLSearchParams({ query: code });
    const urlQuery = `${url}?${queryParams.toString()}`;

    const response = await fetch(urlQuery);
    if (!response.ok) {
        return [];
    }

    try {
        const responseData = await response.json();
        return responseData;
    } catch (error) {
        console.error("Error parsing response data:", error);
        return [];
    }
}

async function GetPropertiesByID(promptID: string): Promise<any> {
    try {
        const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
        const path: string = '/weaviate/propertiesbyid';
        const url: string = `${baseUrl}${path}`;

        const queryParams = new URLSearchParams({ id: promptID });
        const urlQuery = `${url}?${queryParams.toString()}`;

        const response = await fetch(urlQuery);
        if (!response.ok) {
            throw new Error("Failed to fetch data");
        }

        const data = await response.json();
        return data;
    } catch (error : any ) {
        throw new Error("Failed to retrieve data: " + error.message);
    }
}

async function fetchPromptCount(functionName: string): Promise<number | string> {

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const promptCountPath: string = '/weaviate/promptcount';
    const url: string = `${baseUrl}${promptCountPath}`;

    const queryParams = new URLSearchParams({ query: functionName });
    const urlQuery = `${url}?${queryParams.toString()}`;

    const response = await fetch(urlQuery);
    if (!response.ok) {
        return "0";
    }

    const data = await response.json();
    return data;
}

function OutputResponseVote(response: any) {

    const outputWindow = vscode.window.createOutputChannel('Response');
    outputWindow.show(true);
    outputWindow.append("The instruct used for this prompt: " + response.instruct + "\n\n");
    outputWindow.append(response.hasResponse);

    const options = [
        { title: `👍 Upvote` },
        { title: `👎 Downvote` }
    ];
    async function voteForPrompt(response: any) {
        const selection = await vscode.window.showInformationMessage('Vote for this prompt:', ...options);
        if (selection) {
            if (selection.title && selection.title.startsWith('👍')) {
                try {
                    Vote(response.id, true);
                    vscode.window.showInformationMessage("Upvote selected");
                } catch (error: any) {
                    vscode.window.showErrorMessage("Failed to upvote: " + error.message);
                }
            } else if (selection.title && selection.title.startsWith('👎')) {
                Vote(response.id, false);
                vscode.window.showInformationMessage("Downvote selected");
            }
        }
    }

    voteForPrompt(response);
}

async function showResponse(isBestResponse:boolean) {
    try {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return;
        }

        const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
        const functionCode = activeEditor.document.getText(selectedFunctionRange);

        const responseList = await getResponseList(functionCode);
        if (responseList.length === 0) {
            vscode.window.showErrorMessage("No response found.");
            return;
        }

        const firstResponseID = responseList[0];
        const responseText = await GetPropertiesByID(firstResponseID);
        if (!responseText) {
            vscode.window.showErrorMessage("Failed to retrieve response.");
            return;
        }

        const outputWindow = vscode.window.createOutputChannel(isBestResponse ? 'Best Response' : 'Random Response');
        outputWindow.show(true);
        outputWindow.append("The instruct used for this prompt: " + responseText.instruct + "\n\n");
        outputWindow.append(responseText.hasResponse);

        remainingResponseList = responseList;

        const options = [
            { title: `👍 Upvote` },
            { title: `👎 Downvote` }
        ];
        const selection = await vscode.window.showInformationMessage('Vote for this prompt:', ...options);
        if (selection) {
            if (selection.title.startsWith('👍')) {
                try {
                    Vote(firstResponseID, true);
                    vscode.window.showInformationMessage("Upvote selected");
                } catch (error: any) {
                    vscode.window.showErrorMessage("Failed to upvote: " + error.message);
                }
            } else if (selection.title.startsWith('👎')) {
                Vote(firstResponseID, false);
                vscode.window.showInformationMessage("Downvote selected");
            }
        }
    } catch (error: any) {
        vscode.window.showErrorMessage("Error: " + error.message);
    }
}

let disposableNextResponse = vscode.commands.registerCommand('modernizer-vscode.showNextResponse', async () => {

    remainingResponseList = remainingResponseList.slice(1);
    await showNextResponse(remainingResponseList);
});

let disposableBest = vscode.commands.registerCommand('modernizer-vscode.showBestResponse', async () => {
    await showResponse(true);
});

let disposableRandom = vscode.commands.registerCommand('modernizer-vscode.showRandomResponse', async () => {
    await showResponse(false);
});

async function showNextResponse(remainingResponseList: string[]) {

    let id = remainingResponseList[0];
    GetPropertiesByID(id)
        .then((response) => {
            OutputResponseVote(response);
        });
}

vscode.commands.registerCommand('modernizer-vscode.openGitHubRepo', (url) => {
    vscode.env.openExternal(vscode.Uri.parse(url));
});

export function activate(context: vscode.ExtensionContext) {
    // Register the CodeLens provider
    context.subscriptions.push(vscode.languages.registerCodeLensProvider('*', new CodelensProvider()));
    // Add disposables to context subscriptions
    context.subscriptions.push(disposableBest);
    context.subscriptions.push(disposableRandom);
    context.subscriptions.push(disposableNextResponse);
}

export function deactivate() {}
