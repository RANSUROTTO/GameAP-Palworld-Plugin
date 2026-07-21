<template>
  <div v-if="server?.game_id !== 'palworld'" class="pw-empty">{{ trans('not_palworld') }}</div>
  <div v-else class="pw-page">
    <div class="pw-notice" :class="connected ? 'ok' : 'warn'">
      <div><strong>{{ connected ? 'Palworld REST API 已连接' : 'Palworld REST API 未连接' }}</strong><small>凭据仅由 GameAP WASM 后端保存和调用，不会返回到浏览器。</small></div>
      <button class="pw-btn" :disabled="busy" @click="refreshAll">刷新</button>
    </div>

    <div v-if="false" class="pw-card pw-config">
      <h3>首次连接设置</h3>
      <input v-model="config.base_url" placeholder="http://host.docker.internal:8212/v1/api">
      <input v-model="config.username" placeholder="admin">
      <input v-model="config.password" type="password" placeholder="Palworld AdminPassword">
      <button class="pw-btn primary" @click="saveConfig">安全保存并连接</button>
    </div>

    <div v-if="error" class="pw-error">{{ error }}</div>
    <div v-if="notice" class="pw-success">{{ notice }}</div>
    <template v-if="configured">
      <div class="pw-kpis">
        <div class="pw-card"><small>在线玩家</small><strong>{{ players.length }}</strong></div>
        <div class="pw-card"><small>服务器 FPS</small><strong>{{ metric('serverfps', 'server_fps') }}</strong></div>
        <div class="pw-card"><small>运行时间</small><strong>{{ metric('uptime', 'uptime') }}</strong></div>
        <div class="pw-card"><small>世界天数</small><strong>{{ metric('days', 'days') }}</strong></div>
      </div>
      <div class="pw-tabs">
        <button v-for="item in tabs" :key="item.id" :class="{active:tab===item.id}" @click="tab=item.id">{{ item.label }}</button>
      </div>
      <div v-if="error" class="pw-error">{{ error }}</div>

      <div v-if="tab==='overview'" class="pw-grid">
        <div class="pw-card"><h3>服务器信息</h3><dl><dt>名称</dt><dd>{{ info.servername || server.name }}</dd><dt>版本</dt><dd>{{ info.version || '—' }}</dd><dt>世界 GUID</dt><dd class="mono">{{ info.worldguid || '—' }}</dd><dt>说明</dt><dd>{{ info.description || '—' }}</dd></dl></div>
        <div class="pw-card"><h3>连接状态</h3><p>{{ connected ? 'GameAP、PalServer 与 REST API 工作正常。' : '等待 REST API 响应。' }}</p><small>自动刷新间隔：15 秒</small></div>
      </div>

      <div v-if="tab==='players'" class="pw-card"><h3>玩家名单</h3><div class="table-wrap"><table><thead><tr><th>玩家</th><th>等级</th><th>延迟</th><th>User ID</th><th>位置</th></tr></thead><tbody><tr v-for="p in players" :key="p.userId"><td>{{ p.name }}</td><td>{{ p.level }}</td><td>{{ p.ping }} ms</td><td class="mono">{{ p.userId }}</td><td>{{ p.location_x }}, {{ p.location_y }}</td></tr><tr v-if="!players.length"><td colspan="5">当前没有在线玩家</td></tr></tbody></table></div></div>

      <div v-if="tab==='announce'" class="pw-card narrow"><h3>发送全服公告</h3><textarea v-model="announcement" placeholder="服务器将在 10 分钟后维护"></textarea><button class="pw-btn primary" :disabled="busy || !announcement.trim()" @click="action('announce',{message:announcement})">发送公告</button></div>

      <div v-if="tab==='manage'" class="pw-grid"><div class="pw-card"><h3>在线玩家操作</h3><select v-model="selectedUser"><option value="">选择玩家</option><option v-for="p in players" :key="p.userId" :value="p.userId">{{ p.name }} · {{ p.userId }}</option></select><input v-model="reason" placeholder="操作原因"><div class="actions"><button class="pw-btn" :disabled="busy || !selectedUser" @click="action('kick',{userid:selectedUser,message:reason})">踢出</button><button class="pw-btn danger" :disabled="busy || !selectedUser" @click="action('ban',{userid:selectedUser,message:reason})">封禁</button></div></div><div class="pw-card"><h3>解除封禁</h3><input v-model="unbanUser" placeholder="steam_00000000000000000"><button class="pw-btn" :disabled="busy || !unbanUser.trim()" @click="action('unban',{userid:unbanUser})">解除封禁</button></div></div>

      <div v-if="tab==='world'" class="pw-grid"><div class="pw-card"><h3>世界存档</h3><button class="pw-btn primary" :disabled="busy" @click="action('save')">立即保存世界</button></div><div class="pw-card"><h3>演员快照</h3><button class="pw-btn" :disabled="busy" @click="loadSnapshot">获取快照</button><pre v-if="snapshot">{{ snapshot }}</pre></div><div class="pw-card danger-zone"><h3>关闭服务器</h3><input v-model.number="shutdownWait" type="number" min="0" max="600"><input v-model="shutdownMessage"><div class="actions"><button class="pw-btn" :disabled="busy" @click="action('shutdown',{waittime:shutdownWait,message:shutdownMessage},true)">安全关闭</button><button class="pw-btn danger" :disabled="busy" @click="action('stop',undefined,true)">强制停止</button></div></div></div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { providePluginTrans } from '../../../GameAP/web/plugin-sdk/src/i18n';
import { useIsAdmin } from '../../../GameAP/web/plugin-sdk/src/context';
const props=defineProps<{serverId:number;server:any;pluginId:string}>(); const {trans}=providePluginTrans(props.pluginId); const isAdmin=useIsAdmin();
const base=computed(()=>`/api/plugins/${props.pluginId}/servers/${props.serverId}`); const configured=ref(true),connected=ref(false),error=ref(''),notice=ref(''),busy=ref(false),tab=ref('overview');
const info=ref<any>({}),metrics=ref<any>({}),players=ref<any[]>([]),snapshot=ref(''); const announcement=ref(''),selectedUser=ref(''),reason=ref(''),unbanUser=ref(''); const shutdownWait=ref(30),shutdownMessage=ref('Server will shutdown for maintenance.');
const config=ref({base_url:'http://host.docker.internal:8212/v1/api',username:'admin',password:''}); const tabs=[{id:'overview',label:'服务器'},{id:'players',label:'玩家'},{id:'announce',label:'公告'},{id:'manage',label:'玩家管理'},{id:'world',label:'世界操作'}]; let timer:number|undefined;
async function request(path:string,options:RequestInit={}){
  // Reuse GameAP's authenticated Axios client. Raw fetch() does not include
  // the bearer token used by the panel and therefore receives HTTP 401.
  const axios=(window as any).axios;
  try {
    const response=await axios.request({
      url:`${base.value}/${path}`,
      method:options.method||'GET',
      data:options.body ? JSON.parse(String(options.body)) : undefined,
      headers:{Accept:'application/json','Content-Type':'application/json'}
    });
    return response.data||{};
  } catch (e:any) {
    const message=e?.response?.data?.error||e?.response?.data?.message||e?.message||'Request failed';
    const wrapped:any=new Error(message);
    wrapped.status=e?.response?.status;
    throw wrapped;
  }
}
async function checkConfig(){const c=await request('config'); configured.value=!!c.configured; if(c.base_url)config.value.base_url=c.base_url;if(c.username)config.value.username=c.username}
async function saveConfig(){error.value='';try{await request('config',{method:'POST',body:JSON.stringify(config.value)});config.value.password='';configured.value=true;await refreshAll()}catch(e:any){error.value=e.message}}
async function refreshAll(){if(!configured.value)return;error.value='';try{const [i,m,p]=await Promise.all([request('info'),request('metrics'),request('players')]);info.value=i;metrics.value=m;players.value=p.players||[];connected.value=true}catch(e:any){connected.value=false;error.value=e.message}}
function metric(...keys:string[]){for(const k of keys)if(metrics.value?.[k]!==undefined)return metrics.value[k];return '—'}
async function action(path:string,payload?:any,danger=false){
  error.value='';notice.value='';
  if(path==='announce'&&!String(payload?.message||'').trim()){error.value='请先填写公告内容。';return}
  if((path==='kick'||path==='ban')&&!payload?.userid){error.value='请先从在线玩家列表中选择玩家。';return}
  if(path==='unban'&&!String(payload?.userid||'').trim()){error.value='请先填写需要解封的玩家 User ID。';return}
  if(danger&&!window.confirm('确认执行此危险操作？'))return;
  busy.value=true;
  try{
    const panelAxios=(window as any).axios;
    // The Windows service wrapper is configured to restart PalServer when the
    // game process exits. Coordinate the REST action with GameAP's service
    // stop task so shutdown/stop actually leaves the server offline.
    if(path==='stop'){
      await panelAxios.post(`/api/servers/${props.serverId}/stop`);
      await new Promise(resolve=>setTimeout(resolve,500));
      await request(path,{method:'POST'});
    }else{
      await request(path,{method:'POST',body:payload?JSON.stringify(payload):undefined});
      if(path==='shutdown')await panelAxios.post(`/api/servers/${props.serverId}/stop`);
    }
    if(path==='announce')announcement.value='';
    notice.value=({announce:'公告已发送。',kick:'玩家已踢出。',ban:'玩家已封禁。',unban:'玩家已解封。',save:'世界存档已保存。',shutdown:'安全关服指令已发送。',stop:'强制停止指令已发送。'} as Record<string,string>)[path]||'操作成功。';
    if(path==='shutdown'||path==='stop')connected.value=false;else await refreshAll();
  }catch(e:any){error.value=e.message}finally{busy.value=false}
}
async function loadSnapshot(){
  error.value='';notice.value='';busy.value=true;
  try{const data=await request('game-data');snapshot.value=JSON.stringify(data,null,2);notice.value='世界演员快照已获取。'}
  catch(e:any){error.value=e.status===404?'当前 Palworld 服务端版本未提供世界演员快照接口。':e.message}
  finally{busy.value=false}
}
onMounted(async()=>{if(props.server?.game_id!=='palworld')return;try{await refreshAll()}catch(e:any){error.value=e.message}timer=window.setInterval(refreshAll,15000)});onUnmounted(()=>{if(timer)clearInterval(timer)});
</script>

<style>
.pw-page{padding:16px;display:grid;gap:16px;color:inherit}.pw-notice{display:flex;justify-content:space-between;align-items:center;padding:14px 16px;border:1px solid #d6d3d1;border-radius:10px;background:rgba(120,113,108,.06)}.pw-notice.ok{border-color:#22c55e}.pw-notice.warn{border-color:#f59e0b}.pw-notice small{display:block;margin-top:4px;opacity:.7}.pw-kpis{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px}.pw-card{padding:16px;border:1px solid rgba(120,113,108,.28);border-radius:10px;background:rgba(120,113,108,.04)}.pw-card>small{opacity:.7}.pw-card>strong{display:block;font-size:25px;margin-top:6px}.pw-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.pw-tabs{display:flex;gap:6px;border-bottom:1px solid rgba(120,113,108,.25)}.pw-tabs button{padding:10px 12px;border:0;background:none;color:inherit;cursor:pointer;border-bottom:2px solid transparent}.pw-tabs button.active{border-color:#22c55e;color:#16a34a}.pw-btn{border:1px solid rgba(120,113,108,.35);border-radius:7px;padding:8px 13px;background:transparent;color:inherit;cursor:pointer}.pw-btn.primary{background:#16a34a;color:white;border-color:#16a34a}.pw-btn.danger{background:#dc2626;color:white;border-color:#dc2626}.pw-card input,.pw-card textarea,.pw-card select{box-sizing:border-box;width:100%;margin:6px 0;padding:9px;border:1px solid rgba(120,113,108,.35);border-radius:7px;background:transparent;color:inherit}.pw-card textarea{min-height:110px}.pw-config{display:grid;gap:5px;max-width:720px}.narrow{max-width:720px}.actions{display:flex;gap:8px;margin-top:8px}.danger-zone{grid-column:1/-1;border-color:#ef4444}.pw-error{padding:10px;color:#dc2626;background:rgba(220,38,38,.08);border-radius:7px}.pw-empty{padding:24px;opacity:.7}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:9px;border-bottom:1px solid rgba(120,113,108,.2)}dl{display:grid;grid-template-columns:120px 1fr;gap:9px}dt{opacity:.65}.mono{font-family:ui-monospace,monospace}pre{max-height:260px;overflow:auto;font-size:11px}@media(max-width:800px){.pw-kpis,.pw-grid{grid-template-columns:1fr 1fr}}@media(max-width:520px){.pw-kpis,.pw-grid{grid-template-columns:1fr}}
.pw-success{padding:10px;color:#15803d;background:rgba(34,197,94,.1);border-radius:7px}
.pw-btn:disabled{opacity:.45;cursor:not-allowed}
</style>
