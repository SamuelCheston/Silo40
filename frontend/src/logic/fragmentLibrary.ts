import { useEffect, useState } from "react";
import { GetFragmentLibrary } from "../../wailsjs/go/main/App";
import type { FragmentTemplateMeta } from "./models";

// 碎片内容库（events/fragments.json）的展示元数据：经绑定从后端权威加载一次，
// 模块级缓存供各面板共用。纯展示层，不参与任何判定；加载失败回退裸键，不阻塞面板。
export type FragmentLibrary = Record<string, FragmentTemplateMeta>;

let cache: FragmentLibrary | null = null;
let pending: Promise<FragmentLibrary> | null = null;

function loadLibrary(): Promise<FragmentLibrary> {
    if (cache) return Promise.resolve(cache);
    if (!pending) {
        pending = GetFragmentLibrary()
            .then(list => {
                const m: FragmentLibrary = {};
                (list || []).forEach(f => { m[f.id] = f; });
                cache = m;
                return m;
            })
            .catch(err => {
                pending = null;
                throw err;
            });
    }
    return pending;
}

export function useFragmentLibrary(): FragmentLibrary {
    const [lib, setLib] = useState<FragmentLibrary>(cache || {});
    useEffect(() => {
        let alive = true;
        loadLibrary()
            .then(m => { if (alive) setLib(m); })
            .catch(() => { /* 回退裸键 */ });
        return () => { alive = false; };
    }, []);
    return lib;
}

// 徽章标题：未登记的键（非碎片事实等）回退原键。
export function fragmentTitle(lib: FragmentLibrary, id: string): string {
    return lib[id]?.title || id;
}

// 悬浮正文：标题（键 · 部门）+ 正文，供原生 title 阅读。
export function fragmentTooltip(lib: FragmentLibrary, id: string): string {
    const f = lib[id];
    if (!f) return id;
    return `${f.title}（${f.id} · ${f.department}）\n${f.body_template}`;
}