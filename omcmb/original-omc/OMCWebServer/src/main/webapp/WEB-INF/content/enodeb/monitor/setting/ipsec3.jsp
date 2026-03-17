<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div style="flex: 1 auto;overflow: auto; height: 100%;" id="ipsec_props_div">
	<div class="form-group" >
		<div class="group-title">
			<span class="title-icon"></span><span class="title-text">Basic Setting</span>
		</div>
		<div class="form-wrap">
			<div class="form-item">
				<label class="form-title">Enable</label>
				<div class="form-item-wrap">
					<input name="Enable" class="easyui-combobox" data-options="data:[{value:1,text:'Enabel'},{value:0,text:'Disable'}],editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}" style="width: 300px;">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Tunnel Name</label>
				<div class="form-item-wrap">
					<input name="Tunnel Name" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">IKE Negotiation Destination Port</label>
				<div class="form-item-wrap">
					<input name="IKEPort" class="easyui-combobox" data-options="data:[{value:500,text:'500'},{value:4500,text:'4500'}],editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}" style="width: 300px;">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">leftAuth</label>
				<div class="form-item-wrap">
					<input id="leftAuth" name="leftAuth" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:leftAuthchange,events:{blur:function(e){
							validIpsecInput(e.target);
							validLeftAuth(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">rightAuth</label>
				<div class="form-item-wrap">
					<input name="rightAuth" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							validRightAuth(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Gateway</label>
				<div class="form-item-wrap">
					<input name="Gateway" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							//ipsecIPvalid(e.target);
						}}"/>
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Right Subnet</label>
				<div class="form-item-wrap">
					<input name="Right Subnet" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							ipsecIPortvalidMult(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">leftId</label>
				<div class="form-item-wrap">
					<input name="leftId" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">rightId</label>
				<div class="form-item-wrap">
					<input name="rightId" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">leftCert</label>
				<div class="form-item-wrap">
					<input id="leftCert" name="leftCert" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">secretKey</label>
				<div class="form-item-wrap">
					<input id="secretKey" name="secretKey" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">rightSecretKey</label>
				<div class="form-item-wrap">
					<input name="rightSecretKey" class="easyui-textbox" style="width: 300px;" data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">fragmentation</label>
				<div class="form-item-wrap">
					<input name="fragmentation" class="easyui-combobox" style="width: 300px;" data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">AuthBy</label>
				<div class="form-item-wrap">
					<input name="AuthBy" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Pre Shared Key</label>
				<div class="form-item-wrap">
					<input name="PreSharedKey" type="password" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">leftSourceIp</label>
				<div class="form-item-wrap">
					<input name="leftSourceIp" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">leftSubnet</label>
				<div class="form-item-wrap">
					<input name="leftSubnet" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
		</div>
	</div>

	<div class="form-group last" >
		<div class="group-title">
			<span class="title-icon"></span><span class="title-text">Advance Setting</span>
		</div>
		<div class="form-wrap">
			<div class="form-item">
				<label class="form-title">IKE Encryption</label>
				<div class="form-item-wrap">
					<input name="IKE Encryption" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">IKE DH Group</label>
				<div class="form-item-wrap">
					<input name="IKE DH Group" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">IKE Authentication</label>
				<div class="form-item-wrap">
					<input name="IKE Authentication" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">ESP Encryption</label>
				<div class="form-item-wrap">
					<input name="ESP Encryption" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">ESP DH Group</label>
				<div class="form-item-wrap">
					<input name="ESP DH Group" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">ESP Authentication</label>
				<div class="form-item-wrap">
					<input name="ESP Authentication" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">KeyLife</label>
				<div class="form-item-wrap">
					<input name="KeyLife" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							hourValid(e.target);
							keylifeValid(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">IKELifeTime</label>
				<div class="form-item-wrap">
					<input name="IKELifeTime" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							hourValid(e.target);
							ikelifeValid(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">RekeyMargin</label>
				<div class="form-item-wrap">
					<input name="RekeyMargin" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							hourValid(e.target);
							minTimeValid(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Self Define Keyingtries <input id="sdk_ck_input" type="checkbox"/></label>
				<div class="form-item-wrap">
					<input id="sdk_input" name="Self Define Keyingtries" class="easyui-textbox" disabled style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Dpdaction</label>
				<div class="form-item-wrap">
					<input name="Dpdaction" class="easyui-combobox" style="width: 300px;" data-options="editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Dpddelay</label>
				<div class="form-item-wrap">
					<input name="Dpddelay" class="easyui-textbox" style="width: 300px;"
						   data-options="onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
							hourValid(e.target);
						}}">
				</div>
			</div>
			<div class="form-item">
				<label class="form-title">Left Interface</label>
				<div class="form-item-wrap">
					<input name="Left Interface" class="easyui-combobox" data-options="data:[{value:'WAN(eth2)',text:'eth2'},{value:'PPPOE(pppoe-wan)',text:'pppoe-wan'},{value:'none',text:''}],editable:false,onChange:ipsecOnchange,events:{blur:function(e){
							validIpsecInput(e.target);
						}}"
						   style="width: 300px;">
				</div>
			</div>
		</div>
	</div>
	<div style="display: none">
		<input id="Index" name="Index" class="easyui-textbox" >
	</div>
</div>
<div style="padding-left: 20px; padding-bottom: 10px;">
	<div class="bt-group linkbuttonGroup">
		<a class="linkbutton" onclick="saveIpsec()"><span><%=rb.getString("QueDing")%></span></a>
		<a class="linkbutton linkbutton_nowanna" onclick="openPropsPanel(false);"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
</div>
<script>
	var tb = $('#55A502E49AD4B7632E10F04E271D37CA');
	if($('#55A502E49AD4B7632E10F04E271D37CA').length) tb = $('#55A502E49AD4B7632E10F04E271D37CA'); // V4
	if($('#68648C856AC993F58BB67077DBD8F693').length) tb = $('#68648C856AC993F58BB67077DBD8F693'); // V3

	var ipsecSelectUrl = "";
	var smallCellCode = sessionStorage.getItem("smallCellCode");

	var leftCertCombo = '';
	var leftCertText = '';
	var secretKeyCombo = '';
	var secretKeyText = '';
	var plist = tb.data('props').list;
	ipsecSelectUrl = tb.data('props').url;
	var ipsecSrow = tb.datagrid('getSelected'),ipsecIdField = tb.data('props').idField,ipsecValidStatus = true;

	if(ipsecSrow) {/* 数据id标识 */
	}else{
		ipsecSrow = {};
		ipsecSrow[ipsecIdField] = tb.data('index')*1 + 1;
		plist.map(function(item){
			var name = item.propName || item.name;
			if(item.dftValue) ipsecSrow[name] = item.dftValue;
		});
	}

	var ipsecNames = plist.map(function(item){return item.propName || item.name;});
	plist.map(function(item){/* 初始化dom属性 */
		var dname = item.propName || item.name;
		var input = $('[name="'+dname+'"]');
		input.data('props',item);

		var dpts = input.attr('data-options');
		if(dpts){
			dpts += ',type:\"'+item.type+'\"';
		}else dpts = 'type:\"'+item.type+'\"';

		dpts += ',name:\"'+dname+'\",typeFlag:\"ipsec\"';
		if(item.required == '1'){
			dpts += ',required: true';
		}

		if(item.data) {
			if(dpts){
				dpts += ',data:'+item.data + ',type:\"'+item.type+'\"';
			}else dpts = 'data:'+item.data + ',type:\"'+item.type+'\"';
		}
		// 将maxlength、minlength初始化到options中
		if(item.maxlength){
			dpts += ',maxlength:'+item.maxlength;
		}
		if(item.minlength){
			dpts += ',minlength:'+item.minlength ;
		}
		if(item.min) dpts += ',max:'+item.max;
		if(item.max) dpts += ',min:'+item.min;

		input.attr('data-options',dpts);
		if(ipsecSrow && ipsecSrow[item.name]){
			input.val(ipsecSrow[item.name]);
		}
	});
	//选择方式为 pubkey时,leftCert,secretKey 需要变换成下拉菜单； 反之，不变，为默认状态
	setTimeout(function(){

		try{
			var curLeftAuth = $("#leftAuth").combobox("getValue");

			if(curLeftAuth == 'pubkey'){
				try{
					leftCertCombo = $("#leftCert").combobox('getValue');
				}catch(e){
					leftCertCombo = $("#leftCert").textbox('getValue');
				}
				try{
					secretKeyCombo = $("#secretKey").combobox('getValue');
				}catch(e){
					secretKeyCombo = $("#secretKey").textbox('getValue');
				}

				var params = {
					"smallCellCode":smallCellCode
				};
				$.post(ipsecSelectUrl,params, function (data) {
					var certData = data.cert;
					if(certData){
						var certInfo = certData['InternetGatewayDevice.Ipsec.CertInfo'];

						var certInfoSelect = certInfo.split(',');

						var certSelect =[];//存放需要的下拉值

						certInfoSelect.map(function(item,index){
							certSelect.push({value:item, text:item});
						});

						//secretKey
						var certInfoTwo = certData['InternetGatewayDevice.Ipsec.SecretKeyInfo'];

						var certInfoSelectTwo = certInfoTwo.split(',');

						var certSelectTwo =[];//存放需要的下拉值

						certInfoSelectTwo.map(function(item,index){
							certSelectTwo.push({value:item, text:item});
						});

						$("#leftCert").combobox({
							data:certSelect,
							valueField:"value",
							textFiled:"text"

						})
						//secretKey
						$("#secretKey").combobox({
							data:certSelectTwo,
							valueField:"value",
							textFiled:"text"
						})
					}


				}, "json")
			}else {
				//textbox 类型
				leftCertText = $("#leftCert").textbox('getValue');
				secretKeyText = $("#secretKey").textbox('getValue');
			}

		}catch(e) {
			console.log(e)
		}

		try{
			$('[name="leftAuth"]').prev().blur();
		}catch(e){}
	},100)

	$('#sdk_ck_input').on('change',function(){
		var bool = $(this).is(':checked');

		if(bool) {
			$('#sdk_input').textbox('enable');
		}else {
			$('#sdk_input').textbox('disable');
		}
	});

	$('#ipsecForm input[name]').each(function(n,item){/* 移除不支持的项 */
		var dName = $(item).prop('name');
		if(!ipsecNames.includes(dName)) $(item).parents('.form-item').remove();
	});

	$('#ipsec_props_div .group-title .title-text').click(function(){
		var ctn = $(this).parents('.form-group'),
				title = $(this).parents('.group-title');
		if(ctn.hasClass('extend')){
			title.next().slideDown();
			ctn.removeClass('extend');
		}else{
			title.next().slideUp();
			ctn.addClass('extend')
		}
	});
	function validIpsecInput(el,type){
		var ctn = $(el).parent().prev(),
				fnName = 'textbox',
				opts = {},code='valid';
		Array.from(ctn[0].classList).map(function(item){
			if(item.indexOf('easyui-')>=0) fnName = item.replace('easyui-','');
		});
		opts = ctn[fnName]('options');

		var msges = Render.messages(opts),
				iVal = opts.value;
		if(type) opts.type = type;

		code = _threeRule(iVal, opts);
		var result = msges[code];

		if(result) {
			ctn.parents('.form-item').addClass('invalid');
			ipsecValidStatus = false;
		}
		else ctn.parents('.form-item').removeClass('invalid');
		ctn.parents('.form-item').attr('data-msg',result);

		function _threeRule(iVal, ops) {
			var code = 'valid';
			if (iVal && iVal != ops.originalValue) {
				if(ops.type == 'num' && isNaN(iVal)) return 'range';
				/* 重启校验 */
				bools = [];
				if (ops.reboot == true && iVal != ops.value) {
					if (ops.type == 'num' && iVal - ops.value == 0) {

					} else code = 'reboot';
				}
				/* 大小校验 */
				if (!isNaN(iVal)) {
					if (ops.max||ops.max===0) bools.push(iVal - ops.max <= 0);
					if (ops.min||ops.min===0) bools.push(iVal - ops.min >= 0);
				}
				bools.map(function(bool) {
					if (!bool) code = 'range';
				});
				/* 长度校验 */
				bools = [];
				if (ops.minlength||ops.minlength===0) bools.push(iVal.length >= ops.minlength);
				if (ops.maxlength||ops.maxlength===0) bools.push(iVal.length <= ops.maxlength);
				bools.map(function(bool) {
					if (!bool) code = 'length';
				});
			} else if(ops.required==true && iVal != ops.originalValue ) code = 'required';

			return code;
		}
	}
	function ipsecIPvalid(el){
		var regstr = '^(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|[1-9])\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)$',
				reg = new RegExp(regstr),
				ctn = $(el).parent().prev(),
				value = ctn.textbox('getValue'),
				originVal = ctn.textbox('options').originalValue;

		var ctner = ctn.parents('.form-item');
		if(isValidIP(value)){
		}else if(value && value!=originVal){
			ctner.attr('data-msg','<%=rb.getString("IPDiZhiFeiFa")%>').addClass('invalid');
			ipsecValidStatus = false;
		}
	}
	function ipsecIPortvalid(el){
		var regstr = '^(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|[1-9])\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)$',
				reg = new RegExp(regstr),
				ctn = $(el).parent().prev(),
				value = ctn.textbox('getValue'),
				originVal = ctn.textbox('options').originalValue,
				regInt = /^-?\d+$/;

		var ctner = ctn.parents('.form-item');
		if(value){
			var pValid = true,
					valArr = value.split('/');
			if(!isValidIP(valArr[0])) pValid = false;
			if(!regInt.test(valArr[1])) pValid = false;
			if(pValid){
			}else if(value!=originVal){
				ctner.attr('data-msg','<%=rb.getString("IPDiZhiFeiFa")%>').addClass('invalid');
				ipsecValidStatus = false;
			}
		}
	}
	function ipsecIPortvalidMult(el){
		var regstr = '^(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|[1-9])\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
				+'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)$',
				reg = new RegExp(regstr),
				ctn = $(el).parent().prev(),
				values = ctn.textbox('getValue'),
				originVal = ctn.textbox('options').originalValue,
				regInt = /^-?\d+$/;

		var ctner = ctn.parents('.form-item');
		if(values){
			var vsArr = values.split(','),pValid = true;

			vsArr.map(function(value){
				var valArr = value.split('/');
				if(!isValidIP(valArr[0])) pValid = false;
				if(!regInt.test(valArr[1])) pValid = false;
			});

			if(pValid){
			}else if(values!=originVal){
				ctner.attr('data-msg','<%=rb.getString("IPDiZhiFeiFa")%>').addClass('invalid');
				ipsecValidStatus = false;
			}
		}
	}
	function ipsecOnchange(newV,oldVal){
		var props = $(this).data('props'),
				fnName = '', origVal = '';
		if(props){
			fnName = $.domRenderDefaults[props.type];
			origVal = $(this)[fnName]('options').originalValue;
		}
		if(newV!=='' && newV != origVal){
			$(this).next('span').addClass('form-modified');
		}else {
			$(this).next('span').removeClass('form-modified');
		}
	}
	//leftAuth 下拉值发生变化时，选择pubkey 时，leftCert，secretKey由输入框的形式变为下拉菜单，反之，为默认状态，并赋值；
	function leftAuthchange(newV,oldVal){
		if(newV == 'pubkey'){
			var params = {
				"smallCellCode":smallCellCode
			};
			//根据接口返回的 url,获取下拉菜单会对应值
			$.post(ipsecSelectUrl,params, function (data) {
				var certData = data.cert;
				if(certData){
					var certInfo = certData['InternetGatewayDevice.Ipsec.CertInfo'];

					var certInfoSelect = certInfo.split(',');

					var certSelect =[];//存放需要的下拉值

					certInfoSelect.map(function(item,index){
						certSelect.push({value:item, text:item});
					});

					//secretKey
					var certInfoTwo = certData['InternetGatewayDevice.Ipsec.SecretKeyInfo'];

					var certInfoSelectTwo = certInfoTwo.split(',');

					var certSelectTwo =[];//存放需要的下拉值

					certInfoSelectTwo.map(function(item,index){
						certSelectTwo.push({value:item, text:item});
					});

					$("#leftCert").combobox({
						data:certSelect,
						valueField:"value",
						textFiled:"text",
						value:leftCertCombo

					})
					//secretKey
					$("#secretKey").combobox({
						data:certSelectTwo,
						valueField:"value",
						textFiled:"text",
						value:secretKeyCombo

					})

				}


			}, "json")
		}else {
			try{
				// 不为pubkey时，从下拉选择变为输入框的形式，并赋值
				var ctnOne = $("#leftCert").parents('.form-item-wrap'),
						optsOne = $("#leftCert").combobox('options');
				//optsOne.value= $("#leftCert").textbox('getValue');
				$("#leftCert").combobox('destroy');
				var textboxOne = $('<input id="leftCert" name="leftCert" style="width: 300px;">');
				ctnOne.append(textboxOne);
				textboxOne.textbox(optsOne);

				$("#leftCert").textbox('setValue',leftCertText);
			}catch(e){}

			try{
				//secretKey
				var ctnTwo = $("#secretKey").parents('.form-item-wrap'),
						optsTwo = $("#secretKey").combobox('options');

				$("#secretKey").combobox('destroy');
				var textboxTwo = $('<input id="secretKey" name="secretKey" style="width: 300px;">');
				ctnTwo.append(textboxTwo);
				textboxTwo.textbox(optsTwo);

				$("#secretKey").textbox('setValue',secretKeyText);
			}catch(e){}
		}

	}
	function hourValid(el){
		var reg = /^\d+[s|m|h]{1}$/,
				ctn = $(el).parent().prev(),
				value = ctn.textbox('getValue'),
				originVal = ctn.textbox('options').originalValue,
				regInt = /^\d+$/;

		var ops = ctn.data('props');
		var ctner = ctn.parents('.form-item'),
				inValidMsg = '<%=rb.getString("ShuZiJiaSMH")%>(<%=rb.getString("ZiFuChangDu")%>'+ops.minlength+'-'+ops.maxlength+')';
		if(enbPlatform == '1') {
			reg = /^\d+[s|m|h|d]{1}$/;
			inValidMsg = '<%=rb.getString("ShuZiJiaSMHD")%>(<%=rb.getString("ZiFuChangDu")%>'+ops.minlength+'-'+ops.maxlength+')';
		}
		if(reg.test(value)){
			var bools = [],code='valid';
			if (ops.minlength) bools.push(value.length >= ops.minlength);
			if (ops.maxlength) bools.push(value.length <= ops.maxlength);
			bools.map(function(bool) {
				if (!bool) code = 'length';
			});
			var result = Render.messages(ops)[code];
			if(result){
				ipsecValidStatus = false;
				ctner.attr('data-msg',inValidMsg).addClass('invalid');
			}else{
				ctner.attr('data-msg','').removeClass('invalid');
			}
		}else if(value && value!=originVal){
			ctner.attr('data-msg',inValidMsg).addClass('invalid');
			ipsecValidStatus = false;
		}
	}
	function keylifeValid(el) {
		var ctn = $(el).parent().prev(),
				ctner = ctn.parents('.form-item'),
				value = ctn.textbox('getValue'),
				reg = /^\d+[s|m|h|d]{1}$/,
				tarDom = $('[textboxname="RekeyMargin"]');

		if(tarDom && tarDom.length && reg.test(value) && getEnbType() != '') {
			var tarVal = tarDom.textbox('getValue');
			if(reg.test(tarVal)) {
				var keylifeTime = getSecondsTime(value),
						rekeyTime = getSecondsTime(tarVal),
						distance = keylifeTime - rekeyTime*3;

				if(distance >= 0) {

				}else {
					ipsecValidStatus = false;
					ctner.attr('data-msg','KeyLife and RekeyMargin must meet: KeyLife >= RekeyMargin*3').addClass('invalid');
				}
			}
		}

		try{
			$('[name="IKELifeTime"]').prev().blur();
		}catch(e){}
	}
	function ikelifeValid(el) {
		var ctn = $(el).parent().prev(),
				ctner = ctn.parents('.form-item'),
				value = ctn.textbox('getValue'),
				reg = /^\d+[s|m|h|d]{1}$/,
				tarDom = $('[textboxname="KeyLife"]');

		if(tarDom && tarDom.length && reg.test(value) && getIKEEnbType() != '') {
			var tarVal = tarDom.textbox('getValue');
			if(reg.test(tarVal)) {
				var keylifeTime = getSecondsTime(value),
						rekeyTime = getSecondsTime(tarVal),
						distance = keylifeTime - rekeyTime;

				if(distance >= 0) {

				}else {
					ipsecValidStatus = false;
					ctner.attr('data-msg','IKELifeTime and KeyLife must meet: IKELifeTime >= KeyLife').addClass('invalid');
				}
			}
		}
	}
	function minTimeValid(el) {
		var ctn = $(el).parent().prev(),
				ctner = ctn.parents('.form-item'),
				value = ctn.textbox('getValue'),
				reg = /^\d+[s|m|h|d]{1}$/,
				minTypes = {
					'RTS&QRTB': '5m',
					V3: '3m'
				},
				type = getEnbType();

		if(minTypes[type] && reg.test(value)) {
			var prevTime = getSecondsTime(value),
					minTime = getSecondsTime(minTypes[type]),
					distance = prevTime - minTime;

			if(distance >= 0) {

			}else {
				ipsecValidStatus = false;
				ctner.attr('data-msg','RekeyMargin >= ' + minTypes[type]).addClass('invalid');
			}

			try{
				$('[name="KeyLife"]').prev().blur();
			}catch(e){}
		}
	}
	function getIKEEnbType() {
		var type = '',
				product = settingVue.selectedRow.product;

		if(['RTS','RTD','QRTB','QRTB-CA','QRTB-DC','QRTB-SC'].includes(product)) {
			type = 'RTS&RTD&QRTB';
		}
		if(['QAFB'].includes(product)) {
			type = 'V3';
		}

		return type;
	}
	function getEnbType() {
		var type = '',
				product = settingVue.selectedRow.product;

		if(['RTS','QRTB','QRTB-CA','QRTB-DC','QRTB-SC'].includes(product)) {
			type = 'RTS&QRTB';
		}
		if(['QAFB'].includes(product)) {
			type = 'V3';
		}

		return type;
	}
	function getSecondsTime(time) {
		var units = {
					s: 1,
					m: 60,
					h: 3600,
					d: 86400
				},
				key = time.match(/[smhd]/)[0],
				value = time.match(/\d*/)[0],
				timeUnit = units[key],
				sTime = value*timeUnit;

		return sTime;
	}
	function validLeftAuth(el) {
		try{
			setTimeout(function(){
				var ctn = $(el).parent().prev(),
						ctner = ctn.parents('.form-item'),
						value = ctn.combobox('getValue'),
						tarDom = $('[textboxname="rightAuth"]'),
						cascade = {
							'psk-psk':         { show: ['secretKey'],  hide: ['leftCert','rightSecretKey'] },
							'psk-pubkey':      { show: ['secretKey'],  hide: ['leftCert','rightSecretKey'] },
							'pubkey-psk':      { show: ['leftCert','secretKey','rightSecretKey'], hide: [] },
							'pubkey-pubkey':   { show: ['leftCert','secretKey'], hide: ['rightSecretKey'] },
							'eap-aka-psk':     { show: ['secretKey','rightSecretKey'], hide: ['leftCert'] },
							'eap-aka-pubkey':  { show: ['leftCert','secretKey'], hide: ['rightSecretKey'] },
							'eap-aka-eap-aka': { show: [], hide: ['leftCert','secretKey','rightSecretKey'] },
							'psk-eap-aka':     {show: [], hide: []},
							'pubkey-eap-aka':  {show: [], hide: []}
						};
	
				// 级联隐藏和显示
				if(tarDom && tarDom.length) {
					var tarVal = tarDom.combobox('getValue'),
							key = value + '-' +tarVal,
							showItems = cascade[key].show,
							hideItems = cascade[key].hide;
	
					showItems.map(function(code){
						$('[textboxname="'+code+'"]').parents('.form-item').show();
					});
					hideItems.map(function(code){
						$('[textboxname="'+code+'"]').parents('.form-item').hide();
					});
				}
			},100);
		}catch(e){}
	}
	function validRightAuth(el) {
		try{
			setTimeout(function(){
				var ctn = $(el).parent().prev(),
						ctner = ctn.parents('.form-item'),
						value = ctn.combobox('getValue'),
						tarDom = $('[textboxname="leftAuth"]'),
						cascade = {
							'psk-psk':         { show: ['secretKey'],  hide: ['leftCert','rightSecretKey'] },
							'psk-pubkey':      { show: ['secretKey'],  hide: ['leftCert','rightSecretKey'] },
							'pubkey-psk':      { show: ['leftCert','secretKey','rightSecretKey'], hide: [] },
							'pubkey-pubkey':   { show: ['leftCert','secretKey'], hide: ['rightSecretKey'] },
							'eap-aka-psk':     { show: ['secretKey','rightSecretKey'], hide: ['leftCert'] },
							'eap-aka-pubkey':  { show: ['leftCert','secretKey'], hide: ['rightSecretKey'] },
							'eap-aka-eap-aka': { show: [], hide: ['leftCert','secretKey','rightSecretKey'] },
							'psk-eap-aka':     {show: [], hide: []},
							'pubkey-eap-aka':  {show: [], hide: []}
						};
	
				// 级联隐藏和显示
				if(tarDom && tarDom.length) {
					var tarVal = tarDom.combobox('getValue'),
							key = tarVal + '-' + value,
							showItems = cascade[key].show,
							hideItems = cascade[key].hide;
	
					showItems.map(function(code){
						$('[textboxname="'+code+'"]').parents('.form-item').show();
					});
					hideItems.map(function(code){
						$('[textboxname="'+code+'"]').parents('.form-item').hide();
					});
	
					if(value == 'eap-aka') {
						tarDom.combobox('setValue', value);
						cascade['eap-aka-eap-aka'].hide.map(function(code){
							$('[textboxname="'+code+'"]').parents('.form-item').hide();
						});
					}
				}
			},100);
		}catch(e){}
	}

	function saveIpsec(){
		ipsecValidStatus = true;
		$('#ipsec_props_div input').blur();
		if(!ipsecValidStatus) return;

		var datas = Render.getJspDatas($('#ipsecForm')),
				type = $('#ipsecForm').attr('operateType');
		if(isEmptyJson(datas)){
			$.messager.alert('info','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
		}else{
			addJspDatasToTable($('#ipsecForm'),tb,type);
		}
		var opts = tb.data('props'),
				hasRows = tb.datagrid('getRows');
		if(hasRows.length>=opts.max) {
			tb.parents('.form-item-wrap').find('.form-tb-title>.group-operations').hide();
		}
		if(hasRows.length>=opts.min) {
			hasRows.map(function(rowItem,idx){
				rowItem._remove = true;
				tb.datagrid('updateRow',{
					index: idx,
					row: rowItem
				});
			})
		}
	}
</script>