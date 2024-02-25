import * as vscode from 'vscode';
import fetch from 'node-fetch';

export function DisplayVoting(promptId: string) {
    const options: vscode.MessageItem[] = [
        { title: `👍 Upvote` },
        { title: `👎 Downvote` }
    ];

    async function voteForPrompt(promptId: string) {
        const selection = await vscode.window.showInformationMessage('Vote for this prompt:', ...options);
        if (selection) {
            if (selection.title?.startsWith('👍')) {
                try {
                    await Vote(promptId, true);
                    vscode.window.showInformationMessage("Upvote selected");
                } catch (error: any) {
                    vscode.window.showErrorMessage("Failed to upvote: " + error.message);
                }
            } else if (selection.title?.startsWith('👎')) {
                await Vote(promptId, false);
                vscode.window.showInformationMessage("Downvote selected");
            }
        }
    }

    voteForPrompt(promptId);
}

export async function Vote(id: string, Upvote: boolean) {
    const requestBody = { id }; // Send the ID in the request body

    let uri = `http://192.168.10.163:8080/vote?upvote=${Upvote}`;

    try {
        const response = await fetch(uri, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! Status: ${response.status}`);
        }
    } catch (error: any) {
        throw new Error("Failed to send POST request: " + error.message);
    }
}
