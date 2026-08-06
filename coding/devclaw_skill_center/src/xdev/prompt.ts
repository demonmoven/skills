import * as readline from 'node:readline';

export type Agent = 'trae-cn' | 'trae' | 'coco' | 'cc' | 'cdx';

export async function promptAgent(): Promise<Agent> {
  console.log('Select coding agent:');
  console.log('  1) Trae CN (trae-cn)');
  console.log('  2) Trae (trae)');
  console.log('  3) Coco / TRAE CLI (coco)');
  console.log('  4) Claude Code (cc)');
  console.log('  5) Codex (cdx)');

  return new Promise((resolve) => {
    const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
    rl.question('Choice [1/2/3/4/5]: ', (answer) => {
      rl.close();
      const choice = answer.trim().toLowerCase();
      if (choice === '1' || choice === 'trae-cn') {
        resolve('trae-cn');
      } else if (choice === '2' || choice === 'trae') {
        resolve('trae');
      } else if (choice === '3' || choice === 'coco') {
        resolve('coco');
      } else if (choice === '4' || choice === 'cc') {
        resolve('cc');
      } else if (choice === '5' || choice === 'cdx') {
        resolve('cdx');
      } else {
        console.error('Invalid choice');
        process.exit(1);
      }
    });
  });
}
