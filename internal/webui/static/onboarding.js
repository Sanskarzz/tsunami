// Typed, rule-checked setup conversation. This is deliberately not labelled a
// live LLM. Protected values never become chat messages or API explorer bodies.
export function createOnboardingUI({api,act,message,page,openEngagement,element,details,time,pretty,actionTable,ruleText}) {
  const $=id=>document.getElementById(id);
  let current=null, editing=false, accessEditor=false, connected=false, locked=false, stopping=false, generation=0;
  let applications=[], reviewedPreflight='', formStage=0;
  const reviewKey=()=>current?current.id+'/'+current.revision+'/'+current.preflight_digest:'';
  const fields=['name','environment','frontend','reference','api','auth','identity','owner','peer','rule','fields'];
  const privateFields=value=>value.split(',').map(x=>x.trim()).filter(Boolean).map(x=>x.startsWith('/')?x:'/'+x.replaceAll('~','~0').replaceAll('/','~1'));
  function showStage(stage, focus=false) {
    formStage=stage;
    for(const panel of document.querySelectorAll('[data-setup-stage]')) panel.hidden=Number(panel.dataset.setupStage)!==stage;
    $('setup-stage-caption').textContent=stage===0?'STEP 1 OF 5 / APPLICATION':'STEP 2 OF 5 / INTEGRATION';
    markStage(stage);
    if(focus) document.querySelector(`[data-setup-stage="${stage}"] input, [data-setup-stage="${stage}"] select`)?.focus();
  }
  function markStage(stage) {
    for(const step of document.querySelectorAll('[data-setup-stage-label]')) {
      if(Number(step.dataset.setupStageLabel)===stage) step.setAttribute('aria-current','step'); else step.removeAttribute('aria-current');
    }
  }
  function lock(on,busy) {
    connected=on;locked=busy;
    $('new-setup').disabled=!on||busy||stopping;
    $('setup-next-step').disabled=!on||busy||stopping;
    $('setup-back-step').disabled=busy||stopping;
    $('setup-use-lab').disabled=!on||busy||stopping||!applications.some(app=>app.id==='vampi-vulnerable');
    $('setup-form').setAttribute('aria-busy',String(busy));
    $('save-setup').disabled=!on||busy||stopping;
    $('refresh-setup').disabled=!on||!current||busy||stopping;
    $('edit-setup').disabled=!on||!current||busy||stopping||current.state==='stopped'||current.state==='validating'||!!current.engagement_id;
    $('update-setup-access').disabled=$('edit-setup').disabled||current?.details.auth_method!=='bearer'||!current?.details.api_origin;
    $('stop-setup').disabled=!on||!current||stopping||current.state==='stopped';
    $('validate-setup').disabled=!on||!current||busy||stopping||Date.parse(current.expires_at)<=Date.now()||current.validation_attempts>=3||!$('setup-preflight-approved').checked||reviewedPreflight!==reviewKey()||!['ready_for_validation','validation_failed','interrupted'].includes(current.state);
    $('prepare-setup').disabled=!on||busy||stopping||!current||(!current.engagement_id&&(Date.parse(current.expires_at)<=Date.now()||current.state!=='ready_for_approval'));
    for (const b of document.querySelectorAll('.setup-open,.setup-submit')) b.disabled=!on||busy||stopping;
  }
  function apply(data,expectedGeneration=generation) {
    if(expectedGeneration!==generation)return;
    if(current&&current.id===data.id&&current.revision>data.revision)return;
    if(!current||current.id!==data.id||current.revision!==data.revision||current.preflight_digest!==data.preflight_digest){reviewedPreflight='';$('setup-preflight-approved').checked=false;}
    current=data;render();
  }
  async function load(id) {
    const g=++generation;
    const data=await api('GET','/v1/onboarding/'+id);
    if(g!==generation)return;
    current=null;editing=false;accessEditor=false;apply(data,g);page('setup');
  }
  function render() {
    $('setup-current').hidden=!current;
    $('setup-form').hidden=!!current&&!editing;
    $('cancel-edit-setup').hidden=!editing;
    if(!current){showStage(formStage);$('setup-question').replaceChildren();$('setup-messages').replaceChildren();$('setup-preflight-approved').checked=false;reviewedPreflight='';renderSource();lock(connected,locked);return;}
    const d=current.details;
    const stage=['ready_for_approval','assessment_created'].includes(current.state)?4:['ready_for_validation','validation_failed','interrupted','validating'].includes(current.state)?3:2;
    if(!editing) markStage(stage);
    $('setup-title').textContent=d.name;
    $('setup-state').textContent=current.state;
    const next={needs_input:'Supply the missing prerequisites below. Saving answers makes no target request.',unsupported:'This authentication adapter is unavailable. No validation or assessment can start.',ready_for_validation:'Review and explicitly allow the four legitimate reads below. This is separate from assessment approval.',validating:'Prerequisite validation is active. Stop remains a separate action.',ready_for_approval:'Four legitimate reads passed. Review a separate assessment scope before approving any security probe.',assessment_created:'A saved assessment draft exists. Open its scope; approval and Start remain separate.',validation_failed:'Readiness was not established. Inspect the recorded reason, correct access/details and explicitly review another validation.',interrupted:'A previous validation is uncertain. Nothing was replayed. Inspect state before approving another attempt.',expired:'Setup permission expired. No readiness or execution permission is inferred.',stopped:'Setup permission is revoked. Cancellation and assessment cleanup are separate observations.'};
    $('setup-next').textContent=next[current.state]||'Read the stored setup state; no readiness is inferred.';
    details($('setup-summary'),[['Environment',d.environment],['API target',d.api_origin||'Awaiting API origin'],['Protected access',current.access_ready?'Configured for this origin; current session health unknown':'Not configured'],['Intended rule',ruleText(d.expected_access)],['Setup permission expires',time(current.expires_at)],['Last validation',current.validation_code|| (current.validated_at?'Passed at '+time(current.validated_at)+'; current session health unknown':'Not established')]]);
    details($('setup-facts'),[['Setup',current.id+' · revision '+current.revision],['Frontend',d.frontend_url],['API origin',d.api_origin||'Not supplied'],['API addresses',(d.approved_addresses||[]).join('\n')||'Not resolved/approved'],['Authentication',d.auth_method||'Not known'],['Protected access',current.access_ready?'Assigned to this origin; not shown':'Not assigned'],['Baseline validation',current.validated_at?time(current.validated_at):(current.validation_code||'Not established')],['Preflight requests',current.validation_requests+' / 4 this attempt · '+current.validation_attempts+' / 3 attempts'],['Setup expiry',time(current.expires_at)],['Advisor mode','Checked workflow · no external model calls']]);
    $('setup-messages').replaceChildren();
    for(const m of current.messages||[]){const row=element('div',undefined,'setup-message '+(m.role==='user'?'user':'assistant'));row.append(element('span',m.role==='user'?'You':'Setup assistant','eyebrow'),element('p',m.text),element('small',time(m.at)));$('setup-messages').append(row);}
    renderQuestion();
    $('setup-question-card').hidden=!current.question&&!accessEditor;
    $('setup-validation-progress').textContent=`Recorded baseline activity: ${current.validation_requests} of 4 request reservations this attempt · ${current.state}. ${current.validation_code || ''}`;
    const canValidate=['ready_for_validation','validation_failed','interrupted','validating'].includes(current.state)&&!!d.api_origin&&current.access_ready;
    $('setup-validation').hidden=!canValidate;
    details($('setup-preflight-summary'),[['Target',d.api_origin],['Approved addresses',(d.approved_addresses||[]).join('\n')],['Permission expires',time(current.expires_at)],['Grant','Four GETs; at most 15 seconds; at least 1 second between reads; no redirects or cross-user probe'],['Current attempt',current.validation_requests+' recorded target requests · '+current.validation_attempts+' of 3 attempts used']]);
    actionTable($('setup-preflight-actions'),[['owner',d.identity_path,'Verify account A identity'],['peer',d.identity_path,'Verify account B identity'],['owner',d.owner_path,"Read A's own record"],['peer',d.peer_path,"Read B's own record"]]);
    $('setup-preflight').textContent=pretty({api_origin:d.api_origin,approved_addresses:d.approved_addresses,method:'GET',reads:[{account:'owner',path:d.identity_path},{account:'peer',path:d.identity_path},{account:'owner',path:d.owner_path},{account:'peer',path:d.peer_path}],limits:{requests:4,deadline_seconds:15,min_interval_ms:1000},grant_digest:current.preflight_digest,validation_code:current.validation_code||undefined});
    $('setup-ready').hidden=!(current.state==='ready_for_approval'||current.engagement_id);
    $('prepare-setup').textContent=current.engagement_id?'Open saved assessment scope →':'Open the assessment scope for approval →';
    lock(connected,locked);
  }
  function input(form,label,type='text',value='') {
    const id='setup-q-'+form.children.length;
    const l=element('label',label);l.htmlFor=id;
    const i=element('input');i.id=id;i.type=type;i.autocomplete='off';i.spellcheck=false;i.value=value;
    i.maxLength=type==='password'?4096:256;i.required=true;form.append(l,i);return i;
  }
  function select(form,label,options) {
    const l=element('label',label),i=element('select');
    i.id='setup-q-'+form.children.length; l.htmlFor=i.id;
    for(const [value,text] of options){const option=element('option',text);option.value=value;i.append(option);}
    form.append(l,i);return i;
  }
  function renderQuestion() {
    const node=$('setup-question');node.replaceChildren();
    const q=accessEditor?{field:'access',text:'Replace the two dedicated bearer test credentials. This invalidates prior readiness and makes no target requests.'}:current.question;
    if(!q||['stopped','expired','validating','assessment_created'].includes(current.state))return;
    const form=element('form',undefined,'setup-answer');
    form.dataset.question=q.field;
    form.append(element('h3','Your input is needed'),element('p',q.text,'hint'));
    let build,action='answers',caption='Send this information';
    if(q.field==='api_origin'){
      const i=input(form,'Backend API origin','url');i.placeholder='https://api.example.test';build=()=>({value:i.value.trim()});
    }else if(q.field==='addresses'){
      const c=element('input');c.type='checkbox';c.required=true;const l=element('label',undefined,'check');l.append(c,document.createTextNode('Allow a bounded DNS lookup for '+current.details.api_origin+'. No API request or credentials will be sent.'));form.append(l);action='resolve';build=()=>({confirm:c.checked});caption='Resolve candidate addresses';
    }else if(q.field==='auth_method'||q.field==='unsupported_auth'){
      const i=select(form,'Authentication integration',[['','Choose a method'],['bearer','Two dedicated bearer tokens'],['password','Username/password · adapter not implemented'],['sso','SSO/MFA · adapter not implemented']]);i.required=true;build=()=>({value:i.value});
    }else if(q.field==='routes'){
      const identity=input(form,'GET current-user path','text',current.details.identity_path),owner=input(form,"Account A's legitimate resource path",'text',current.details.owner_path),peer=input(form,"Account B's legitimate resource path",'text',current.details.peer_path);build=()=>({identity_path:identity.value.trim(),owner_path:owner.value.trim(),peer_path:peer.value.trim()});
    }else if(q.field==='access_rule'){
      const i=select(form,"Intended access when B reads A's resource",[['','Choose the business rule'],['deny','Access must be denied'],['allow','Intentionally shared, including the selected proof'],['filtered','Share public record, omit private fields']]);i.required=true;build=()=>({value:i.value});
    }else if(q.field==='protected_fields'){
      const i=input(form,'Private synthetic field names');i.placeholder='secret or /private_note';build=()=>({fields:privateFields(i.value)});
    }else if(q.field==='test_data'){
      const c=element('input');c.type='checkbox';c.required=true;const l=element('label',undefined,'check');l.append(c,document.createTextNode('These dedicated accounts and records contain only disposable synthetic test data.'));form.append(l);build=()=>({confirm:c.checked});
    }else if(q.field==='access'){
      form.append(element('p','Protected access form. These values are encrypted in the state store; the mounted key is outside that database. No real password/SSO adapter is available.','hint'));
      const owner=input(form,'Account A bearer token','password'),peer=input(form,'Account B bearer token','password');owner.minLength=32;peer.minLength=32;action='access';caption='Save protected access';
      build=()=>{const body={owner:owner.value.trim(),peer:peer.value.trim()};owner.value='';peer.value='';return body;};
    }else{return;}
    const submit=element('button',caption,'button primary setup-submit');submit.type='submit';form.append(submit);
    form.addEventListener('submit',event=>{event.preventDefault();void act(async()=>{
      const g=generation,id=current.id;
      const body={revision:current.revision,...(action==='answers'?{question_id:q.id}:{}),...build()};
      const data=await api('POST','/v1/onboarding/'+id+'/'+action,body);accessEditor=false;apply(data,g);
    });});
    node.append(form);
  }
  function formDetails() {
    const value={name:$('setup-name').value.trim(),environment:$('setup-environment').value.trim(),frontend_url:$('setup-frontend').value.trim(),authorization_ref:$('setup-reference').value.trim(),source_application:$('setup-source').value,
      api_origin:$('setup-api').value.trim(),auth_method:$('setup-auth').value,identity_path:$('setup-identity').value.trim(),owner_path:$('setup-owner').value.trim(),peer_path:$('setup-peer').value.trim(),expected_access:$('setup-rule').value,protected_fields:privateFields($('setup-fields').value),synthetic_only:$('setup-synthetic').checked,
      approved_addresses:editing&&current?current.details.approved_addresses:undefined};
    if(value.source_application&&!editing){for(const key of ['api_origin','auth_method','identity_path','owner_path','peer_path','expected_access','protected_fields'])delete value[key];}
    return value;
  }
  function renderSource(){
    const app=applications.find(app=>app.id===$('setup-source').value);
    $('setup-source-info').hidden=!app;
    $('setup-extra').hidden=!!app&&!editing;
    if(app)details($('setup-source-details'),[['Existing API',app.origin],['Approved addresses',app.approved_addresses.join('\n')],['Intended rule',ruleText(app.expected_access)],['Reuse','Existing lab credentials, routes and test facts are reused by the backend. A frontend URL does not change this API target. Saving creates no network permission.']]);
  }
  function clearForm() {
    for(const id of fields)$('setup-'+id).value='';$('setup-environment').value='dev';$('setup-source').value='';$('setup-source').disabled=false;$('setup-synthetic').checked=false;
  }
  function start(source=''){
    if(!connected||locked||stopping)return;
    generation++;current=null;editing=false;accessEditor=false;clearForm();formStage=0;
    // A shortcut only chooses a returned registration. Saving/validation still require explicit action.
    if(typeof source==='string'&&applications.some(app=>app.id===source))$('setup-source').value=source;
    render();page('setup');
  }
  $('new-setup').addEventListener('click',start);
  $('setup-source').addEventListener('change',renderSource);
  $('setup-use-lab').addEventListener('click',()=>{
    if(!applications.some(app=>app.id==='vampi-vulnerable'))return;
    $('setup-source').value='vampi-vulnerable';renderSource();
  });
  $('setup-next-step').addEventListener('click',()=>{
    const inputs=document.querySelectorAll('[data-setup-stage="0"] input');
    for(const input of inputs)if(!input.reportValidity())return;
    showStage(1,true);
  });
  $('setup-back-step').addEventListener('click',()=>showStage(0,true));
  $('setup-form').addEventListener('invalid',event=>{
    const panel=event.target.closest('[data-setup-stage]');
    if(panel)showStage(Number(panel.dataset.setupStage));
  },true);
  $('update-setup-access').addEventListener('click',()=>{accessEditor=true;renderQuestion();lock(connected,locked);});
  $('setup-form').addEventListener('submit',event=>{event.preventDefault();void act(async()=>{
    const d=formDetails(),g=generation;
    const data=editing?await api('POST','/v1/onboarding/'+current.id+'/details',{revision:current.revision,details:d}):await api('POST','/v1/onboarding',d);
    editing=false;current=null;apply(data,g);message('Setup saved. Review the assistant questions; no security assessment has started.');
  });});
  $('edit-setup').addEventListener('click',()=>{
    if(!current)return;editing=true;showStage(0);const d=current.details;
    for(const [id,key] of [['name','name'],['environment','environment'],['frontend','frontend_url'],['reference','authorization_ref'],['api','api_origin'],['auth','auth_method'],['identity','identity_path'],['owner','owner_path'],['peer','peer_path'],['rule','expected_access']])$('setup-'+id).value=d[key]||'';
    $('setup-source').value=d.source_application||'';$('setup-source').disabled=true;$('setup-fields').value=(d.protected_fields||[]).join(', ');$('setup-synthetic').checked=d.synthetic_only;render();
  });
  $('cancel-edit-setup').addEventListener('click',()=>{editing=false;render();});
  $('refresh-setup').addEventListener('click',()=>void act(async()=>{const g=generation;apply(await api('GET','/v1/onboarding/'+current.id),g);}));
  $('setup-preflight-approved').addEventListener('input',()=>{reviewedPreflight=$('setup-preflight-approved').checked?reviewKey():'';lock(connected,locked);});
  $('validate-setup').addEventListener('click',()=>void act(async()=>{
    if(!$('setup-preflight-approved').checked||reviewedPreflight!==reviewKey())throw new Error('Review and approve the current revision of the exact four-read grant.');
    const id=current.id,g=generation,revision=current.revision,digest=current.preflight_digest;
    $('setup-preflight-approved').checked=false;
    let observing=true,timer;
    async function observe(){if(!observing||g!==generation)return;try{apply(await api('GET','/v1/onboarding/'+id),g);}catch{/* The validation response/explicit refresh decides its outcome. */}if(observing)timer=setTimeout(observe,700);}
    timer=setTimeout(observe,400);
    message('Validation requested; awaiting server observations. Stop remains available. This is not a security probe.');
    try {apply(await api('POST','/v1/onboarding/'+id+'/validate',{revision,digest,confirm:true}),g);}finally{observing=false;clearTimeout(timer);}
  }));
  $('prepare-setup').addEventListener('click',()=>void act(async()=>{
    const g=generation,id=current.id;
    const data=await api('POST','/v1/onboarding/'+id+'/prepare',{revision:current.revision});apply(data,g);
    if(g===generation)await openEngagement(data.engagement_id);
  }));
  $('stop-setup').addEventListener('click',async()=>{
    if(!current||stopping)return;const id=current.id,g=generation;stopping=true;lock(connected,locked);
    try {apply(await api('POST','/v1/onboarding/'+id+'/stop',{}),g);message('Setup Stop acknowledged. Permission is revoked; cancellation and any assessment workload cleanup are separate observations.');}
    catch(error){message('Setup Stop was NOT acknowledged. Inspect saved state; do not assume work ended. '+error.message,true);}
    finally{stopping=false;lock(connected,locked);}
  });
  return {
    lock,start,
    lists(sessions,apps){
      applications=apps.filter(app=>app.metadata?.registration==='startup');
      const selected=$('setup-source').value;$('setup-source').replaceChildren();const first=element('option','New integration · ask for missing facts');first.value='';$('setup-source').append(first);
      for(const app of applications){const option=element('option','Reuse lab: '+app.title+' · '+app.id);option.value=app.id;$('setup-source').append(option);}if([...$('setup-source').options].some(o=>o.value===selected))$('setup-source').value=selected;renderSource();
      $('setup-history').replaceChildren();
      for(const item of sessions.sort((a,b)=>b.created_at.localeCompare(a.created_at))){const b=element('button',undefined,'history-row setup-open');b.type='button';b.dataset.setup=item.id;const text=element('span',item.details.name);text.append(element('small',item.id));b.append(text,element('span',item.state,'badge'));b.addEventListener('click',()=>void act(()=>load(item.id)));$('setup-history').append(b);}
      if(!sessions.length)$('setup-history').textContent='No saved setups.';lock(connected,locked);
    },
    reset(){generation++;formStage=0;current=null;editing=false;accessEditor=false;stopping=false;clearForm();$('setup-history').textContent='Not connected.';$('setup-messages').replaceChildren();$('setup-question').replaceChildren();$('setup-preflight').textContent='';$('setup-preflight-approved').checked=false;render();}
  };
}
