import { createRequire } from 'node:module';
import type { Language } from '../types.js';

const require = createRequire(import.meta.url);
const Graph = require('graphology');

export interface FileNodeAttr { type: 'file'; lang: Language; loc: number; }
export interface SymbolNodeAttr { type: 'symbol'; file: string; line: number; kind: 'function' | 'class' | 'type' | 'module'; name: string; }
export interface ModuleNodeAttr { type: 'module'; lang: Language; loc: number; files: string[]; }
export interface ConfigNodeAttr { type: 'config'; kind: 'dockerfile' | 'workflow' | 'pkg'; path: string; }
export interface CommitNodeAttr { type: 'commit'; sha: string; time: number; author: string; }

export class UnifiedGraph {
  private g: any;

  constructor() {
    this.g = new Graph({ multi: true, type: 'mixed' });
  }

  addFile(path: string, attrs: Omit<FileNodeAttr, 'type'>): void {
    if (!this.g.hasNode(path)) this.g.addNode(path, { type: 'file', ...attrs });
  }

  addImport(from: string, to: string): void {
    const key = `imp:${from}->${to}`;
    if (!this.g.hasEdge(key)) {
      this.g.addDirectedEdgeWithKey(key, from, to, { kind: 'imports' });
    }
  }

  addSymbol(id: string, attrs: Omit<SymbolNodeAttr, 'type'>): void {
    if (!this.g.hasNode(id)) {
      this.g.addNode(id, { type: 'symbol', ...attrs });
      const edgeKey = `def:${attrs.file}->${id}`;
      if (!this.g.hasEdge(edgeKey)) this.g.addDirectedEdgeWithKey(edgeKey, attrs.file, id, { kind: 'defines' });
    }
  }

  addCoChange(a: string, b: string, weight: number): void {
    const [x, y] = a < b ? [a, b] : [b, a];
    const key = `cc:${x}|${y}`;
    if (this.g.hasEdge(key)) {
      this.g.setEdgeAttribute(key, 'weight', weight);
    } else {
      this.g.addUndirectedEdgeWithKey(key, x, y, { kind: 'co-changes', weight });
    }
  }

  coChangeWeight(a: string, b: string): number {
    const [x, y] = a < b ? [a, b] : [b, a];
    const key = `cc:${x}|${y}`;
    return this.g.hasEdge(key) ? (this.g.getEdgeAttribute(key, 'weight') as number) : 0;
  }

  files(): string[] {
    return this.g.filterNodes((_: string, attrs: any) => attrs.type === 'file').sort();
  }

  symbolsOf(file: string): Array<Omit<SymbolNodeAttr, 'type'>> {
    return this.g.filterNodes((_: string, attrs: any) => attrs.type === 'symbol' && attrs.file === file)
      .map((id: string) => {
        const a = this.g.getNodeAttributes(id) as SymbolNodeAttr;
        const { type: _t, ...rest } = a;
        return rest;
      });
  }

  importsOf(file: string): string[] {
    return this.g.outNeighbors(file).filter((n: string) => {
      const a = this.g.getNodeAttributes(n);
      return a.type === 'file';
    });
  }

  importersOf(file: string): string[] {
    return this.g.inNeighbors(file).filter((n: string) => {
      const a = this.g.getNodeAttributes(n);
      return a.type === 'file';
    });
  }

  raw(): any { return this.g; }
}
