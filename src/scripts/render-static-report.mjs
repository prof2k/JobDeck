import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const marker = '__JOB_DECK_REPORT_DATA__';
const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const templatePath = resolve(scriptDirectory, '../static-report/template.html');
const [sourceArgument, outputArgument] = process.argv.slice(2);

if (!sourceArgument || !outputArgument) {
  throw new Error(
    'Usage: node src/scripts/render-static-report.mjs <report.json> <report.html>',
  );
}

const sourcePath = resolve(sourceArgument);
const outputPath = resolve(outputArgument);
const [template, source] = await Promise.all([
  readFile(templatePath, 'utf8'),
  readFile(sourcePath, 'utf8'),
]);
const reportData = JSON.parse(source);

if (
  typeof reportData !== 'object'
  || reportData === null
  || typeof reportData.report !== 'object'
  || reportData.report === null
  || !Array.isArray(reportData.jobs)
) {
  throw new Error('Report data must contain a report object and jobs array.');
}

if (!template.includes(marker)) {
  throw new Error('Standalone report template is missing its data marker.');
}

const escapedData = JSON.stringify(reportData).replace(
  /[<>&\u2028\u2029]/g,
  (character) => `\\u${character.charCodeAt(0).toString(16).padStart(4, '0')}`,
);
const rendered = template.replace(marker, escapedData);

await mkdir(dirname(outputPath), { recursive: true });
await writeFile(outputPath, rendered, 'utf8');

console.log(`Rendered standalone report: ${outputPath}`);
