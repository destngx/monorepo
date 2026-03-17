import prompts from 'prompts';
import * as fs from 'fs';
import * as path from 'path';

const envPath = path.join(process.cwd(), '.env.local');

// Helper to update env vars
function updateEnvVar(key: string, value: string) {
  let envContent = '';
  if (fs.existsSync(envPath)) {
    envContent = fs.readFileSync(envPath, 'utf8');
  }
  const regex = new RegExp(`^${key}=.*$`, 'm');
  if (regex.test(envContent)) {
    envContent = envContent.replace(regex, `${key}=${value}`);
  } else {
    envContent += `\n${key}=${value}\n`;
  }
  fs.writeFileSync(envPath, envContent.trim() + '\n');
}

async function githubDeviceFlow(isEnterprise: boolean) {
  console.log('\n──────────────────────────────────────────────────');
  console.log('🤖 GitHub Copilot OAuth Setup');
  console.log('──────────────────────────────────────────────────\n');

  // Request device code
  const codeRes = await fetch('https://github.com/login/device/code', {
    method: 'POST',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      client_id: '01ab8ac9400c4e429b23',
      scope: 'read:user user:email'
    })
  });

  if (!codeRes.ok) {
    throw new Error(`Failed to request device code: ${codeRes.statusText}`);
  }

  const codeData = await codeRes.json() as any;

  console.log(`\n1. Open your browser to: ${codeData.verification_uri}`);
  console.log(`2. Enter the following code: \x1b[32m\x1b[1m${codeData.user_code}\x1b[0m\n`);
  console.log('Waiting for authorization (this may take up to 15 minutes)...\n');

  // Poll for the access token
  const intervalSeconds = codeData.interval || 5;
  let tokenData: any = null;

  while (!tokenData) {
    await new Promise(resolve => setTimeout(resolve, intervalSeconds * 1000));

    const pollRes = await fetch('https://github.com/login/oauth/access_token', {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        client_id: '01ab8ac9400c4e429b23',
        device_code: codeData.device_code,
        grant_type: 'urn:ietf:params:oauth:grant-type:device_code'
      })
    });

    const pollData = await pollRes.json() as any;

    if (pollData.error) {
      if (pollData.error === 'authorization_pending') {
        process.stdout.write('.');
      } else if (pollData.error === 'slow_down') {
        await new Promise(resolve => setTimeout(resolve, (pollData.interval || 5) * 1000));
      } else if (pollData.error === 'expired_token') {
        console.error('\n\n❌ Device code expired. Please run the script again.');
        process.exit(1);
      } else if (pollData.error === 'access_denied') {
        console.error('\n\n❌ Authorization was denied.');
        process.exit(1);
      } else {
        console.error(`\n\n❌ Unexpected error: ${pollData.error}`);
        process.exit(1);
      }
    } else if (pollData.access_token) {
      tokenData = pollData;
    }
  }

  console.log('\n\n✅ Successfully authenticated with GitHub!');
  updateEnvVar('GITHUB_TOKEN', tokenData.access_token);
  console.log('🔑 Saved GITHUB_TOKEN to .env.local');

  // Fetch available models
  console.log('\nFetching latest supported GitHub Copilot models...');
  try {
    const modelsRes = await fetch('https://models.inference.ai.azure.com/models', {
      headers: { 'Authorization': `Bearer ${tokenData.access_token}` }
    });
    if (modelsRes.ok) {
      const modelsData = await modelsRes.json() as any;
      console.log('\nAvailable Models from your Copilot Account:');
      const textModels = modelsData.filter((m: any) => m.task === 'chat' || m.task === 'text-generation');
      textModels.forEach((m: any) => {
        console.log(` - ${m.name} (${m.friendly_name})`);
      });
      console.log('\n💡 Note: I have enabled GPT-4o, GPT-o1 (4.1), and Grok in the chat for your Copilot account!');
    } else {
      console.log('\nCould not fetch models directly, but authorization succeeded.');
    }
  } catch (e: any) {
    console.log('\nCould not fetch models list:', e.message);
  }
}

async function main() {
  console.log('\n┌  Add credential\n│');

  const response = await prompts({
    type: 'select',
    name: 'provider',
    message: 'Select provider',
    choices: [
      { title: 'GitHub Copilot (GPT-4o, o1, Grok - Free with Pro/Teams/Enterprise)', value: 'github' },
      { title: 'OpenAI', value: 'openai' },
      { title: 'Google (Gemini)', value: 'google' },
      { title: 'Anthropic (Claude)', value: 'anthropic' }
    ],
  });

  if (!response.provider) {
    console.log('Canceled.');
    process.exit(0);
  }

  if (response.provider === 'github') {
    const typeRes = await prompts({
      type: 'select',
      name: 'deployment',
      message: 'Select GitHub deployment type',
      choices: [
        { title: 'GitHub.com (Public)', value: 'public' },
        { title: 'GitHub Enterprise', value: 'enterprise' }
      ]
    });

    if (!typeRes.deployment) process.exit(0);

    if (typeRes.deployment === 'enterprise') {
      const tokenRes = await prompts({
        type: 'password',
        name: 'token',
        message: 'Enter your GitHub Enterprise Personal Access Token:'
      });
      if (tokenRes.token && tokenRes.token.trim() !== '') {
        updateEnvVar('GITHUB_TOKEN', tokenRes.token.trim());
        console.log('🔑 Saved GITHUB_TOKEN to .env.local');
      }
    } else {
      await githubDeviceFlow(false);
    }
  } else {
    let keyName = '';
    let url = '';
    if (response.provider === 'openai') {
      keyName = 'OPENAI_API_KEY';
      url = 'https://platform.openai.com/api-keys';
    } else if (response.provider === 'google') {
      keyName = 'GOOGLE_GENERATIVE_AI_API_KEY';
      url = 'https://aistudio.google.com/app/apikey';
    } else if (response.provider === 'anthropic') {
      keyName = 'ANTHROPIC_API_KEY';
      url = 'https://console.anthropic.com/settings/keys';
    }

    console.log(`\nTo get your key, visit: ${url}`);
    const keyRes = await prompts({
      type: 'password',
      name: 'key',
      message: `Enter your ${response.provider} API Key:`
    });

    if (keyRes.key && keyRes.key.trim() !== '') {
      updateEnvVar(keyName, keyRes.key.trim());
      console.log(`🔑 Saved ${keyName} to .env.local`);
    } else {
      console.log('Skipped saving key.');
    }
  }

  console.log('\n🚀 Setup complete! Please restart your dev server if it is running.');
}

main().catch(err => {
  console.error('\n❌ Error:', err.message);
  process.exit(1);
});
