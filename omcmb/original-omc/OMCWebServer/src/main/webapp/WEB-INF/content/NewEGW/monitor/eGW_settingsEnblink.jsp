<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>

	#egwSettingLinkPage	.el-input__suffix{
		height: 26px;
		display: flex;
		align-items: center;
	}
	#egwSettingLinkPage	.el-form-item__label{
		font-size: 14px;
	}
	#egwSettingLinkPage .settingContentBoxCls{
		padding: 15px 0px 20px 40px;
	}
	#egwSettingLinkPage .titleStyML{
		margin-bottom: 20px;
	}
	#egwSettingLinkPage .el-form-item__error{
		padding-top: 0px;
		top:35px;
	}
	#egwSettingLinkPage .enbIdBoxCls{
		margin-bottom: 15px;
	}
	#egwSettingLinkPage .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 10px; 
	}
	#egwSettingLinkPage .linkListBoxCls{
		width: 800px;
		margin-left: 25px;
	}
	#egwSettingLinkPage .enbIdMmeListHeader{
		display: flex;
		justify-content: space-between;
		margin-bottom: 10px;
	}
	#egwSettingLinkPage .enbIdMmeListHeader .el-icon::before{
		font-size: 18px;
	}
	#egwSettingLinkPage .linkListTableBoxCls{
		height: 280px;
	}
	#egwSettingLinkPage .settingContentBoxCls  .plmnTitleCls{
		height: 100%;
		width: 100%;
		font-size: 12PX;
		line-height: 26px;
		padding: 0px 5px;
		color: rgba(0, 0, 0, 0.8);
		background-color: #F5F7FA;
		border: 1px solid #DCDFE6;
		border-radius: 4px 0px 0px 4px;
		box-sizing: border-box;
	}
	#egwSettingLinkPage .allowMoreInputFieldCls .el-select .el-input{
		width: 120px;
	}
	#egwSettingLinkPage .allowMoreInputFieldCls .el-input__prefix{
		left: -1px;
	}
	#egwSettingLinkPage .allowMoreInputFieldCls .el-input--prefix .el-input__inner{
		padding-left: 50px;
	}
</style>
<div class="flex-ctn" id="egwSettingLinkPage" style="overflow:hidden">
	<el-form ref="settingForm" :model="settingForm" :rules="formRules" label-position="top"  :hide-required-asterisk='true'>
		<div class="settingContentBoxCls">
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("JiBenPeiZhi")%></span>
			</div>
			<!-- 基本信息 -->
			<div class="enbIdBoxCls">
				<el-form-item prop="enbId" :label="idLabel"  label-width="120px" style="margin:0px 0px 10px 25px;">
					<el-input style='width:200px;' :disabled="operationType == 'edit'" v-model="settingForm.enbId"></el-input>
				</el-form-item>
			</div>
			<div class="allowMoreInputBoxCls">
				<div class="allowMoreInputHeadCls">
					<span class="allowMoreInputTitleCls">PLMN/TAC</span>
					<span class="allowMoreInputTipsCls">（Add 4 at most）</span>
				</div>
				<div class="allowMoreInputContentCls">
					<div class="allowMoreInputFieldCls">
						<el-select  v-model="settingForm.plmn" :disabled="operationType == 'edit'" @change="plmnSelectChange">
							<template  slot="prefix">
								<div class="plmnTitleCls">PLMN</div>
							</template>
							<el-option v-for="item in plmnList" :label="item.Hplmn" :value="item.Hplmn" ></el-option>
						</el-select>
						<el-input v-model="settingForm.tac" :disabled="operationType == 'edit'" style="width: 260px;" :placeholder="tacPlaceholder">
							<template  slot="prepend">
								<div>TAC</div>
							</template>
						</el-input>
						<div class="allowMoreInputAddBtnCls" @click="addPlmnAndTac" v-show="plmnAndTacList.length < 4 ">
							<span class="el-icon el-icon-plus"></span>
							<span>Add</span>
						</div>
					</div>
					<div class="allowMoreInputParamsCls">
						<div v-for="(item,index) in plmnAndTacList" class="allowMoreInputParamsItemCls">
							<span style="margin-right:5px;">PLMN：{{item.plmn}}</span>
							<span style="margin-left:5px;">TAC：{{item.tac}}</span>
							<span class="el-icon el-icon-close" style="margin-left:5px;" v-if="operationType !== 'edit'" @click="plmnAndTacListDel(item,index)"></span>
						</div>
					</div>
				</div>
				<div class="allowMoreInputFootCls">
					<p class="inputErrorBoxCls">{{tacErrorMessage}}</p>
				</div>
			</div>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="settingContentBoxCls">
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("LianLuPeiZhi")%></span>
				<span style="color:#999999;margin-left:-5px;">( <%=rb.getString("eGWZuiDuoTianJian16")%> )</span>
			</div>
			<div class="linkListBoxCls">
				<div class="enbIdMmeListHeader">
					<span style="font-size:14px;font-weight:bold;">Link List</span>
					<span class="el-icon el-icon-circle-add" @click="addLink" v-show="linkTableData.length<16"></span>
				</div>
				<div class="linkListTableBoxCls">
					<el-ctable 
						ref="enbIdMmeListTable" 
						:rownumber="true" 
						id="enbIdMmeListTable" 
						:data="linkTableData" 
						:query-params="queryEnbIdMmeParams" 
						height="100%"
						:pagination="false"
						style="border:1px solid #E9E9E9;"
					>
						<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
							<template slot-scope="scope">
								<!--<span class="el-icon el-icon-operation-edit" @click="editLink(scope.row,event)" style="margin-right:15px;"></span>-->
								<span class="el-icon el-icon-operation-delete" @click="delLink(scope.row,event)" ></span>
							</template>
						</el-table-column>
						<el-table-column label='Local IP' min-width="120" prop="LocalAddrIp" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="deviceType == 'enb'" label='Second Local IP' min-width="120" prop="SecondLocalAddrIp" show-overflow-tooltip></el-table-column>
						<el-table-column label='Local Port' min-width="120" prop="LocalAddrPort" show-overflow-tooltip></el-table-column>
						<el-table-column label='Remote IP' min-width="120" prop="RemoteAddrIp" show-overflow-tooltip></el-table-column>
						<el-table-column label='Remote Port' min-width="120" prop="RemoteAddrPort" show-overflow-tooltip></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
	</el-form>
	<!--链路新增 修改弹窗-->
	<el-dialog id="linkDialog" class="gnbConfigAddDialog" :title="linkDialogTitle" :visible.sync="showLinkDialog" width="1000" :close-on-click-modal="false" top="15vh" @close="linkDialogClose" :append-to-body="true">
		<el-form  :model="linkDialogForm" ref="linkDialogForm" :rules="linkDialogRules" label-position="top">
			<el-form-item prop="localIp" style="min-width:400px;" label="Local IP" class='validate-item'>
				<el-input v-model="linkDialogForm.localIp" @change="localIpChange">
					<template slot="append">Support IPv4 or IPv6</template>
				</el-input>
			</el-form-item>
			<el-form-item prop="localPort" label="Local Port" style="min-width:400px;" class='validate-item'>
				<el-input v-model="linkDialogForm.localPort" @change="localPortChange">
					<template slot="append"><%=rb.getString("FanWei")%>:1024-65535</template>
				</el-input>
			</el-form-item>
			<el-form-item prop="remoteIp" label="Remote IP" style="min-width:400px;" class='validate-item'>
				<el-input v-model="linkDialogForm.remoteIp">
					<template slot="append">Support IPv4 or IPv6</template>
				</el-input>
			</el-form-item>
			<el-form-item prop="remotePort" label="Remote Port" style="min-width:400px;" class='validate-item'>
				<el-input v-model="linkDialogForm.remotePort">
					<template slot="append"><%=rb.getString("FanWei")%>:1024-65535</template>
				</el-input>
			</el-form-item>
			<el-form-item prop="secondLocalIp" label="Second Local IP" style="min-width:400px;" class='validate-item'>
				<el-input v-model="linkDialogForm.secondLocalIp">
					<template slot="append">Support IPv4 or IPv6</template>
				</el-input>
			</el-form-item>
			
		</el-form>
		<div slot="footer" class="importFooter">
			<el-button type="primary" @click="linkDialogSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="linkDialogClose"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script type="text/javascript">
	
	var egwSettingLinkVue = new Vue({
		el: '#egwSettingLinkPage',
		data(){
			var vm = this;
			var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
			var validateEnbId = function(rule,value,callback) { // 校验ENBID
					if(this.deviceType == 'enb'){
						if(value){
							if(vm.isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=1048575){
								vm.$refs.settingForm.validateField('plmn');
								callback();
							}else{
								callback(new Error('<%=rb.getString("FanWei")%>:0-1048575'))
							}
						}else{
							callback(new Error('<%=rb.getString("BiTian")%>'))
						}
					}else{
						if(value){
							callback();
						}else{
							callback(new Error('<%=rb.getString("BiTian")%>'))
						}
					}
						
				},
				validatePlmn = function(rule,value,callback) { // 校验ENBID
					var EnodeBIdAndPlmn = vm.settingForm.enbId +''+ value;
					if(vm.enbIdAndPlmnList.includes(EnodeBIdAndPlmn) == true){
						if(EnodeBIdAndPlmn == vm.oldIdAndPlmn){
							callback();
						}else{
							callback(new Error('<%=rb.getString("PlmnYiPeiZhi")%>'))
						}
					}else{
						callback();
					}
				},
				validateLocalIp = function(rule,value,callback) { // 校验LocalIp
					if(value){
						if(vm.isValidIP(value) || vm.isIPv6(value)) {
							if(vm.linkDialogForm.localPort){
								var str = value +''+ vm.linkDialogForm.localPort
								if(vm.ipAndPortIsOnly(str,vm.linkTableData)){
									callback()
								}else{
									callback('Local IP and Port already exist')
								}
							}else{
								callback()
							}
						}else {
							callback('<%=rb.getString("IPGeShiBuDui")%>');
						}
					}else{
						callback(new Error('<%=rb.getString("BiTian")%>'))
					}
				},
				validateLocalPort = function(rule,value,callback) { // 校验LocalPort
					if(value){
						if(vm.isNumeric(value)&& parseInt(value)>=1024 && parseInt(value)<=65535){
							if(vm.linkDialogForm.localIp){
								var str = vm.linkDialogForm.localIp+ '' + value;
								if(vm.ipAndPortIsOnly(str,vm.linkTableData)){
									callback()
								}else{
									callback('Local IP and Port already exist')
								}
							}else{
								callback()
							}
						}else{
							callback(new Error('<%=rb.getString("FanWei")%>:1024-65535'))
						}
					}else{
						callback(new Error('<%=rb.getString("BiTian")%>'))
					}
				},
				validateRemoteIp = function(rule,value,callback) { // 校验RemoteIp 
					if(value){
						if(vm.isValidIP(value) || vm.isIPv6(value)) {
							callback()
						}else {
							callback('<%=rb.getString("IPGeShiBuDui")%>');
						}
					}else{
						callback(new Error('<%=rb.getString("BiTian")%>'))
					}
				},
				validateRemotePort = function(rule,value,callback) { // 校验RemotePort
					if(value){
						if(vm.isNumeric(value)&& parseInt(value)>=1024 && parseInt(value)<=65535){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%>:1024-65535'))
						}
					}else{
						callback(new Error('<%=rb.getString("BiTian")%>'))
					}
				},
				validateSecondLocalIp = function(rule,value,callback) { // 校验RemoteIp 
					if(value){
						if(vm.isValidIP(value) || vm.isIPv6(value)) {
							callback()
						}else {
							callback('<%=rb.getString("IPGeShiBuDui")%>');
						}
					}else{
						callback()
					}
				};
			return {
				settingForm:{
					enbId:'',
					tac:'',
					plmn:'',
				},
				formRules:{
					enbId:[
						{validator: validateEnbId}
					],
					plmn:[
						{validator: validatePlmn}
					],
				},
				readonly:true,
				queryEnbIdMmeParams:{
					timeZone:timeZone,
				},
				linkDialogTitle:'',
				showLinkDialog:false,
				tacErrorMessage:'',
				linkDialogForm:{
					localIp:'',
					secondLocalIp:'',
					localPort:'',
					remoteIp:'',
					remotePort:'',
				},
				linkDialogRules:{
					localIp:[
						{validator: validateLocalIp}
					],
					secondLocalIp:[
						{validator: validateSecondLocalIp}
					],
					localPort:[
						{validator: validateLocalPort}
					],
					remoteIp:[
						{validator: validateRemoteIp}
					],
					remotePort:[
						{validator: validateRemotePort}
					],
				},
				plmnAndTacList:[],
				tacStr:'',
				plmnList:[],
				linkTableData:[],
				delEnbLinkList:[],
				oldIdAndPlmn:'',
				enbIdAndPlmnList:[],
				operationType:'add',
				deviceType:'',
			}
		},
		computed: {
			idLabel(){
				return this.deviceType == 'enb' ? 'eNodeB ID' : 'gNodeB ID'
			},
			tacPlaceholder(){
				return this.deviceType == 'enb' ? '<%=rb.getString("TACGeShiTiShi")%>' : 'Range:1-16777215,except 16777214'
			}
		},
		watch: {
			plmnAndTacList(){
				var data = this.plmnAndTacList,
					tacList = [];
				data.map((item)=>{
					tacList.push(item.tac)
				});
				this.tacStr = tacList.join(',');
			}
		},
		methods: {
			// 初始化
			init(opType,row,deviceType){
				var vm = this, enbList = [], 
					enbIdAndPlmnList=[];
				if(deviceType == 'enb'){
					enbList = egw4GSignalingGatewayPage.enbIdMmeListTableData || [];
					vm.plmnList = egw4GSignalingGatewayPage.enbPlmnList || [];
				}else{
					enbList = egw5GSignalingGatewayPage.gnbIdMmeListTableData || [];
					vm.plmnList = egw5GSignalingGatewayPage.gnbPlmnList || [];
				}
				enbList.map((item)=>{
					enbIdAndPlmnList.push(item.enodebId +''+ item.Hplmn);
				})
				vm.operationType = opType;
				vm.deviceType = deviceType;
				vm.editRow = row;
				vm.enbIdAndPlmnList = enbIdAndPlmnList;
				
				if(opType !="add"){
					vm.settingForm.enbId = row.enodebId;
					vm.settingForm.plmn = row.Hplmn;
					if(row.Tac){
						var plmnAndTacData = [],
						tacList = row.Tac.split(',');
						tacList.map((item)=>{
							var data={
									plmn:row.Hplmn,
									tac:item
								};
							plmnAndTacData.push(data);
						})
						vm.plmnAndTacList = plmnAndTacData;
					}
					vm.oldIdAndPlmn = row.enodebId +''+ row.Hplmn;
					vm.linkTableData = row.enbLinkListInfo || [];
				}else{
					if(vm.plmnList.length>0){
						vm.settingForm.plmn = vm.plmnList[0].Hplmn;
					}else{
						vm.settingForm.plmn = '';
					}
					
				}
			},
			// 链路新增
			addLink(){
				var vm = this;
				vm.linkDialogTitle = '<%=rb.getString("XinZeng")%>'
				vm.showLinkDialog = true;
			},
			// 链路修改
			editLink(){
				var vm = this;
				vm.linkDialogTitle = '<%=rb.getString("XiuGai")%>'
			},
			// 链路删除
			delLink(row){
				var vm = this;

				if(row.configIndex){
					vm.delEnbLinkList.push(row.configIndex);
				}
				vm.linkTableData = vm.linkTableData.filter((items)=>{
					return (items.LocalAddrIp +''+items.LocalAddrPort) != (row.LocalAddrIp +''+ row.LocalAddrPort)
				})
			},
			// PLMN 下拉改变事件
			plmnSelectChange(val){
				var vm = this;

				if(vm.editRow.Hplmn == val){
					vm.linkTableData = vm.editRow.enbLinkListInfo;
				}else{
					vm.linkTableData  = [];
				}
				vm.plmnAndTacList = [];
			},
			// PLMN/TAC 添加事件
			addPlmnAndTac(){
				var vm = this,
					val = vm.settingForm.tac,
					data={
						plmn:vm.settingForm.plmn,
						tac:vm.settingForm.tac
					};
				if(!data.plmn){
					vm.tacErrorMessage = 'Please select PLMN';
					return
				}
				if(val){
					if(vm.deviceType == 'enb'){
						if(vm.isNumeric(val)&&parseInt(val)>=1 && parseInt(val)<=65535){
							var isExist =  vm.plmnAndTacList.some(item =>item.tac == val);
							if(isExist){
								vm.tacErrorMessage = '<%=rb.getString("YiCunZai")%>'
							}else{
								if(parseInt(val) == 65534){
									vm.tacErrorMessage = '<%=rb.getString("TACGeShiTiShi")%>'
								}else{
									vm.plmnAndTacList.push(data);
									vm.settingForm.tac = '';
									vm.tacErrorMessage = '';
								}
							}
						}else{
							vm.tacErrorMessage = '<%=rb.getString("TACGeShiTiShi")%>'
						}
					}else{
						if(vm.isNumeric(val)&&parseInt(val)>=1 && parseInt(val)<=16777215){
							var isExist =  vm.plmnAndTacList.some(item =>item.tac == val);
							if(isExist){
								vm.tacErrorMessage = '<%=rb.getString("YiCunZai")%>'
							}else{
								if(parseInt(val) == 16777214){
									vm.tacErrorMessage = 'Range:1-16777215,except 16777214'
								}else{
									vm.plmnAndTacList.push(data);
									vm.settingForm.tac = '';
									vm.tacErrorMessage = '';
								}
							}
						}else{
							vm.tacErrorMessage = 'Range:1-16777215,except 16777214'
						}
					}
						
				}
				
			},
			// PLMN/TAC  删除事件
			plmnAndTacListDel(item,index){
				var vm = this;
				vm.plmnAndTacList.splice(index,1);
			},
			localIpChange(val){
				var vm = this;
				if(val && vm.linkDialogForm.localPort){
					vm.$refs.linkDialogForm.validateField('localPort');
				}
			},
			localPortChange(val){
				var vm = this;
				if(val && vm.linkDialogForm.localIp){
					vm.$refs.linkDialogForm.validateField('localIp');
				}
			},
			// 弹窗提交事件
			linkDialogSubmit(){
				var vm = this,
					urls = '',
					params={
						LocalAddrIp:vm.linkDialogForm.localIp,
						SecondLocalAddrIp:vm.linkDialogForm.secondLocalIp,
						LocalAddrPort:vm.linkDialogForm.localPort,
						RemoteAddrIp:vm.linkDialogForm.remoteIp,
						RemoteAddrPort:vm.linkDialogForm.remotePort,
						Hplmn:vm.settingForm.plmn,
					};
				if(params.SecondLocalAddrIp == ''){
					delete params.SecondLocalAddrIp
				}
				vm.$refs.linkDialogForm.validate((valid) => {
					if(valid){
						vm.linkTableData.push(params);
						vm.linkDialogClose();
					}else{
						return false;
					}
				})
				
			},
			// 弹窗关闭
			linkDialogClose(){
				var vm = this;
				vm.showLinkDialog = false;
				vm.$refs.linkDialogForm.resetFields();
				
			},
			// 判断ip port组合是否有重复
			ipAndPortIsOnly(str,data){
				var isOnly = true;
				if(data.length==0){
					isOnly = true;
				}
				data.map((item)=>{
					var ipAndPortStr = item.LocalAddrIp + '' + item.LocalAddrPort;
					if(ipAndPortStr == str){
						isOnly = false;
					}
				})
				return isOnly;
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
			// 链路配置提交
			submit(){
				var vm = this;
				vm.$refs.settingForm.validate((valid) => {
					if(valid){
						let params = {},enbLinkListInfo=[];

						var enbList={
							enodebId:vm.settingForm.enbId,
							linkNum:vm.linkTableData.length,
							Hplmn:vm.settingForm.plmn,
							Tac:vm.tacStr,
							enbLinkListInfo:vm.linkTableData,
							delEnbLinkList:vm.delEnbLinkList.join(','),
							isEdit:'false',
						};
						if(vm.operationType != 'add' ){
							var codeList = ['linkNum','Hplmn','Tac'];
							codeList.map((code)=>{
								if(enbList[code] != vm.editRow[code]){
									enbList.isEdit = 'true';
								}	
							})
							if(enbList.delEnbLinkList){
								enbList.isEdit = 'true';
							}
						}else{
							enbList.isEdit = 'true';
						}
						if(vm.plmnAndTacList.length == 0){
							vm.tacErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
							return
						}
						eventBus.$emit("edit-enbList",enbList);
						eventBus.$emit("close-linkSetting");
					}else{
						return false;
					}
				})
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
		},
		created(){},
		mounted(){
			eventBus.$off('link-init').$on('link-init',this.init);
			eventBus.$off('egw-settingLink-ok').$on('egw-settingLink-ok',this.submit);
		}
	});
</script>