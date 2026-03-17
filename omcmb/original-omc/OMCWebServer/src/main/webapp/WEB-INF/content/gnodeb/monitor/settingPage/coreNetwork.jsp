<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbCoreNetworkPage{
	height: 100%;
	width: 100%;
}
#gnbCoreNetworkPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbCoreNetworkPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbCoreNetworkPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbCoreNetworkPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbCoreNetworkPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbCoreNetworkPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbCoreNetworkPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbCoreNetworkPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbCoreNetworkPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbCoreNetworkPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbCoreNetworkPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbCoreNetworkPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbCoreNetworkPage .el-form-item{
	margin-bottom: 20px;
}
#gnbCoreNetworkPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbCoreNetworkPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbCoreNetworkPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
</style>

<div id="gnbCoreNetworkPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			<%=rb.getString("CoreNetwork")%>
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="AMF">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">AMF</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--AMF-->
							<div> 
								<div class="contentTableTitle">
									<div>AMF List</div>
									<div><span class="el-icon el-icon-circle-add" @click="addAMFDialogOpen('','add','AMF')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="amfTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="amfTable" 
										:data="ruleForm.AMFList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='ID' min-width="40" prop="AMF_idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-delete" @click="delAMFList(scope.row,event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='AMF IP' min-width="120" prop="AMF_IP" show-overflow-tooltip></el-table-column>
										<el-table-column label='PLMN ID' min-width="120" prop="AMF_PLMNID" show-overflow-tooltip></el-table-column>
										<el-table-column label='Default' min-width="120" prop="AMF_Default" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='AMFList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.AMFList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="HaloB">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">HaloB</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='HaloB_Enable' style="width:40%;min-width:400px;" label="HaloB Enable" label-width="160px">
									<el-switch v-model="ruleForm.HaloB_Enable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='HaloB_Mode' style="width:40%;min-width:400px;" label="<%=rb.getString("LicenseMoShi")%>" label-width="160px">
									<el-select v-model='ruleForm.HaloB_Mode'>
										<el-option label="Centralized" value="1"></el-option>
										<el-option label="Single" value="2"></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="LGW">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">LGW</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div class="multiPlmnEnableBoxCls">
								<el-form-item prop='LGW_Enable' style="width:100%;min-width:400px;display:flex;" label="LGW" label-width="180px">
									<el-switch v-model="ruleForm.LGW_Enable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
								</el-form-item>
							</div>
							<div style="display:flex;margin-left:16px;flex-wrap: wrap" v-show="ruleForm.LGW_Enable == '1'">
								<el-form-item prop='LGW_Mode' style="width:40%;min-width:400px;" label="LGW Mode" label-width="160px">
									<el-select v-model='ruleForm.LGW_Mode'>
										<el-option label='NAT' value='16'></el-option>
										<el-option label='Router' value='17'></el-option>
										<el-option label='Bridge' value='18'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='LGW_InterfaceBinding' style="width:40%;min-width:400px;" label="LGW Interface Binding" label-width="160px">
									<el-select v-model='ruleForm.LGW_InterfaceBinding'>
										<el-option label='opt' value='opt'></el-option>
										<el-option label='opt0_1' value='opt0_1'></el-option>
									</el-select>
								</el-form-item>
							</div>
							<div class="multiPlmnEnableBoxCls" v-show="ruleForm.LGW_Enable == '1'">
								<el-form-item prop='LGW_IPv4Enable' style="width:100%;min-width:400px;display:flex;" label="IPv4" label-width="180px">
									<el-switch v-model="ruleForm.LGW_IPv4Enable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
								</el-form-item>
							</div>
							<div style="display:flex;margin-left:16px;flex-wrap: wrap" v-show="ruleForm.LGW_Enable == '1' && ruleForm.LGW_IPv4Enable == '1'">
								<el-form-item prop='LGW_IPv4Address' style="width:40%;min-width:400px;" label="IPv4 Address" label-width="160px">
									<el-input v-model.trim='ruleForm.LGW_IPv4Address'></el-input>
								</el-form-item>
								<el-form-item prop='LGW_IPv4SubnetMask' style="width:40%;min-width:400px;" label="Subnet Mask" label-width="160px">
									<el-input v-model.trim='ruleForm.LGW_IPv4SubnetMask'></el-input>
								</el-form-item>
							</div>
							<div class="multiPlmnEnableBoxCls" v-show="ruleForm.LGW_Enable == '1'">
								<el-form-item prop='LGW_IPv6Enable' style="width:100%;min-width:400px;display:flex;" label="IPv6" label-width="180px">
									<el-switch v-model="ruleForm.LGW_IPv6Enable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
								</el-form-item>
							</div>
							<div style="display:flex;margin-left:16px;flex-wrap: wrap" v-show="ruleForm.LGW_Enable == '1' && ruleForm.LGW_IPv6Enable == '1'">
								<el-form-item prop='LGW_IPv6Address' style="width:40%;min-width:400px;" label="IPv6 Address" label-width="160px">
									<el-input v-model.trim='ruleForm.LGW_IPv6Address'></el-input>
								</el-form-item>
								<el-form-item prop='LGW_IPv6PrefixLength' style="width:40%;min-width:400px;" label="Prefix Length" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.LGW_IPv6PrefixLength'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~128,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
	<!-- AMF新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addAMFDialogShow" @close="closeAddAMFDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addAMFDialogForm" :model='addAMFDialogForm' :rules='addAMFDialogRules' label-position="top">     		     			            
			<el-form-item prop='AMF_IP' style="min-width:400px;" label="AMF IP" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addAMFDialogForm.AMF_IP'>
					<template slot="append">Example：1.1.1.1</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='AMF_PLMNID' style="min-width:400px;" label="PLMN ID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addAMFDialogForm.AMF_PLMNID'>
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='AMF_Default' style="min-width:400px;" label="Default" label-width="160px">
				<el-select v-model='addAMFDialogForm.AMF_Default' style="width:60px;padding-top:5px;">
					<el-option label='0' value='0'></el-option>
					<el-option label='1' value='1'></el-option>
				</el-select>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addAMFDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addAMFDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbCoreNetworkPage = new Vue({
	el: '#gnbCoreNetworkPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
			validateAMF_IPAddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				if(vm.tbType !== 'AMF'){
					callback()
				}else{
					if(value === ''){
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}else{
						if(vm.isValidIP(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}
					}
				}
			},
			validateAMF_PLMNID = (rule,value,callback) => {
				var reg = /^[0-9]{5,6}$/
				if(vm.tbType !== 'AMF'){
					callback()
				}else{
					if(value === ''){
						callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}
					}
				}
			},
			validateIPv4address= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				if(this.ruleForm.LGW_IPv4Enable == '1'){
					if(value == '' || value == undefined || value == null){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}
				}else{
					callback();
				}
			},
			validateIPv4SubnetMask = (rule,value,callback) => {
				if(this.ruleForm.LGW_IPv4Enable == '1'){
					if(value == '' || value == undefined || value == null){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}else{
						if(vm.isMask(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
						}
					}
				}else{
					callback();
				}
			},
			validateIPv6address= (rule,value,callback) => {
				if(this.ruleForm.LGW_IPv6Enable == '1'){
					if(value == '' || value == undefined || value == null){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
					}else{
						if(vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
						}
					}
				}else{
					callback();
				}
			},
			validateIPv6PrefixLength = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
				if(this.ruleForm.LGW_IPv6Enable == '1'){
					if(value == '' || value == undefined || value == null){
						callback(new Error(mag))
					}else{
						if(reg.test(value) && value >= min && value <= max){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
				}else{
					callback();
				}
			};
		return {
			activeCollapse:['AMF','HaloB','LGW'],
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
				AMFList:[],

				HaloB_Enable:'0',
				HaloB_Mode:'0',

				LGW_Enable:'0',
				LGW_Mode:'16',
				LGW_InterfaceBinding:'opt',
				LGW_IPv4Enable:'0',
				LGW_IPv4Address:'',
				LGW_IPv4SubnetMask:'',
				LGW_IPv6Enable:'0',
				LGW_IPv6Address:'',
				LGW_IPv6PrefixLength:'',

			},
			rules:{
				LGW_IPv4Address:[
					
					{validator:validateIPv4address,trigger:'blur'}
				],
				LGW_IPv4SubnetMask:[
					{validator:validateIPv4SubnetMask,trigger:'blur'}
				],
				LGW_IPv6Address:[
					{validator:validateIPv6address,trigger:'blur'}
				],
				LGW_IPv6PrefixLength:[
					{validator:validateIPv6PrefixLength,min:0,max:128,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~128,Integer'}
				],
			},
			casts:{
				'62F88DDFDF0BD19C8B5F63397500DA3F':'AMFList',
				'6CD7293B84945FD7BF8CA79DE3DDFD97':'AMF_idx',
				'C77DC824D5272B6B10B7E30E216B88F0':'AMF_IP',
				'D28FB622BE892AEE0B311FB50A49F8CC':'AMF_PLMNID',
				'8BC7F4F244A547F8871154D9164817FF':'AMF_Default',

				'808122E4D7A23811A18E64D117B13660':'HaloB_Enable',
				'DB0E7ADF007CE41C11782D18570F764F':'HaloB_Mode',

				'B519697904D99B7019D7DEA6E7E1143C':'LGW_Enable',
				'4DBA42FC0E944567463DAB376C90BB14':'LGW_Mode',
				'14CF473175FE3C9897D734ABC677E50C':'LGW_InterfaceBinding',
				'4C2E4581CC386C1E1F61299C686CD58D':'LGW_IPv4Enable',
				'7D0D99CB27ED6B85C0E28FE002FC946C':'LGW_IPv4Address',
				'6AA62BE43118630E7AD3CCB195242ED8':'LGW_IPv4SubnetMask',
				'554207B2A9FB16DA5BA32CD87FF4094B':'LGW_IPv6Enable',
				'2F2211C5693915187967D5210BD80EDF':'LGW_IPv6Address',
				'44FFF245F154CC542732FF0FC90B32C8':'LGW_IPv6PrefixLength',
			},
			codeList:[],
			optType:'',
			tbType:'',
			addAMFDialogShow:false,
			addAMFDialogForm:{
				AMF_IP:'',
				AMF_PLMNID:'',
				AMF_Default:'0',
			},
			addAMFDialogRules:{
				AMF_IP:[{required:true,trigger:'blur'},{validator:validateAMF_IPAddress,trigger:'blur'}],
				AMF_PLMNID:[{required:true,trigger:'blur'},{validator:validateAMF_PLMNID,trigger:'blur'}],
			},
		};
	},
	computed: {
		gnbConfigAddDialogTitle(){
			return this.optType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
		},
	},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23002');
		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					cellIndex:'1',
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
			var key = vm.casts[code];
			try{
				if(key){
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'AMF List'){
							vm.ruleForm.AMFList.push(obj);
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					AMFList:[],
					HaloB_Enable:'0',
					HaloB_Mode:'0',

					LGW_Enable:'0',
					LGW_Mode:'NAT',
					LGW_InterfaceBinding:'opt',
					LGW_IPv4Enable:'0',
					LGW_IPv4Address:'',
					LGW_IPv4SubnetMask:'',
					LGW_IPv6Enable:'0',
					LGW_IPv6Address:'',
					LGW_IPv6PrefixLength:'',
				};
			Object.assign(vm.ruleForm,params);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		// 打开新增 AMF弹窗
		addAMFDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addAMFDialogForm,row)
			}
			vm.addAMFDialogShow = true;
		},
		// 新增 AMF提交
		addAMFDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'AMF':'AMFList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addAMFDialogForm).forEach(function(key){
				if(key.substring(0,3) == vm.tbType){
					params[key] = vm.addAMFDialogForm[key]
				}
			})
			if(vm.addAMFDialogForm.operateType){
				params.operateType = vm.addAMFDialogForm.operateType
			}
			vm.$refs.addAMFDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addAMFDialogShow = false;
				}
			})
		},
		// 删除 Amf
		delAMFList(row){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm.AMFList.map(function(item,index){
					if(item.AMF_idx == row.AMF_idx){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm.AMFList,index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm.AMFList = vm.ruleForm.AMFList.filter((items)=>{
						return items.AMF_idx != row.AMF_idx
					})
				}
			})
		},
		closeAddAMFDialog(){
			var vm = this,
				params = {
					AMF_IP:'',
					AMF_PLMNID:'',
					AMF_Default:'0',
				};
			Object.assign(vm.addAMFDialogForm,params);
			vm.$refs.addAMFDialogForm.clearValidate();
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								if(listVal != 'Xn_Status'){
									objs[listKey] = items[listVal]
								}
							}
							objs.cellIndex = '1';
							subList.push(objs)
						})
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
							cellIndex:vm.cellIndex,
							value:field.fieldValue
						}
						params[key] = editData;

					};
				}
			});
			if(vm.ruleForm.LGW_Enable == '0'){
				['LGW_Mode','LGW_InterfaceBinding','LGW_IPv4Enable','LGW_IPv4Address','LGW_IPv4SubnetMask','LGW_IPv6Enable','LGW_IPv6Address','LGW_IPv6PrefixLength'].map((item)=>{
					var key = vm.getNameByProp(item);
					delete params[key]
				})
			}else{
				if(vm.ruleForm.LGW_IPv4Enable == '0'){
					['LGW_IPv4Address','LGW_IPv4SubnetMask'].map((item)=>{
						var key = vm.getNameByProp(item);
						delete params[key]
					})
				}
				if(vm.ruleForm.LGW_IPv6Enable == '0'){
					['LGW_IPv6Address','LGW_IPv6PrefixLength'].map((item)=>{
						var key = vm.getNameByProp(item);
						delete params[key]
					})
				}
			}
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gnbSetting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							eventBus.$emit('close-gnb-settingPage');
						}else{
							vm.$message.error(data["message"])
						}
						$('#gnbSetting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
			eventBus.$emit('close-gnb-settingPage');
		},
		//校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gnbTabSettingVue.changeMain('coreNetwork');
				}else{
					vm.$message.error(data["message"])
				}
			})
		}
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
