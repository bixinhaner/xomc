<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.tabBox{
		margin:25px
	}
	.capabilityBox{
		margin: 10px 0px 0px 30px;
		border:#DCDFE6 solid 1px;
		border-radius: 3px;
		width:93%;
		height:85% !important
	}
	.capabilityBox th,el-table tr{
		background: #FFFFFF;

	}
	.capabilityBox .el-table__header th{
		height: 51px;
		border:none
	}
	.capabilityBox .el-table__body tr{
		height: 60px
	}
	.keyBox .el-input__inner{
		border:none;
		color: #000 !important;
		background-color:#FFFFFF !important
	}
	.keyBox .el-input__inner:hover{
		background-color:#EDF6FF !important
	}
	.keyBox .el-tooltip{
		margin-left: -20px
	}
	.capabilityBox .el-table__body-wrapper, .capabilityBox .el-table__header-wrapper{
		width:96%;
		margin:0 auto
	}
	.el-icon-circle-add:before{
		font-size: 18px;
	}
	.tslide{
		margin:-15% auto;
		position: relative !important
	}
	.modelBox{
		z-index:9
	}
	.modelBox .el-form-item__label{
		line-height: 25px;
		width: 110px
	}
	.modelBox .el-card__body{
		height: auto;
		background:#fff
	}
	.modelBox .form-group{
		margin-bottom: 0px
	}
	.el-icon-close{
		font-size: 14px
	}
	.quickSteting .el-card__footer{
		display: none
	}
	.el-form-item__error{
		margin-left:110px
	}
	.no-data:before{
		content: ''
	}
	.el-table--border td{
		border-right:none
	}
</style>
<!--基站快速设置浮层  -->
<div id="apInfo" class='slidebarPanels flex-ctn panelDefault' style="background:#FFFFFF;top: 0;bottom:0px;left: 0;right:0px;">
    <div class="quickSteting" id="quickSteting" style="height:100%;overflow: auto;">
    		<div style="padding: 15px 0px 0px 30px;">
    			<span style="padding-right: 5px;"><%=rb.getString("KaiGuan")%></span>
    			<el-switch 
    				active-value="1"
    				inactive-value="0"
    				v-model="lteTurboEnable" 
    				@change="apEnableChange"></el-switch>
    		</div>
            <div v-show="lteTurboEnable=='1'" id="capability" style="flex: auto;overflow: auto;flex-direction: column;position:relative;height:100%">
				<div class="tabBox">
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text">AP List</span>
					</div>
					<div  class="circleIcon placeholder-bt" placeholder="<%=rb.getString("TianJia")%>" style='margin-right:50px;top: 20px;'>		
						<span class="el-icon-circle-add el-icon" @click="addInfo"></span>
					</div>
					<el-ctable ref="capability_list" :rownumber="true" id="viewEnb_table" :time='6' :page-size="pageSize" pagination="true" :query-params="params_ap" :height="tabheight" width="90%" url='${ctx}/cell/ap/queryAPInfos.action' class="capabilityBox">
						<el-table-column label='' min-width="30" prop="op">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" ></div>
							</template>
						</el-table-column>
						<el-table-column label='Serial Number' prop="serial_number"></el-table-column>
						<el-table-column label='SSID' prop="ssid"></el-table-column>
						<el-table-column label='Encryption' prop="encryption"></el-table-column>
						<el-table-column label='Key' prop="key">
							<template slot-scope="scope" class='mr30'>
								●●●
							</template>
						</el-table-column>
					</el-ctable>
					<el-cmenu ref="menuActiveFault" :data="menus" @click="clickMenu"></el-cmenu>
					
					
				</div>
				<!--弹窗页面部分 -->
				<el-dialog :title="'<%=rb.getString("TianJia")%>'" :visible.sync="showWindowInfo" ref="slide" :width="sildeWidth" :height="sildeHeight" 
					:close-on-click-modal="false" :url="dialogUrl"  @close="closeSettingInfo" @success="openDialogSuc" :append-to-body="true">
						<div class="modelBox">
							<div class='el-card__body' style="padding:20px;border:none">
								<el-form  label-position="left" :model="settingsInfo" :rules="settingsRules" ref="settingsForm">
									<div class="form-group last" >
										<div class="plr15 ml20">
											<el-form-item label="Serial Number" prop="serial_number">
												<el-input v-model="settingsInfo.serial_number" placeholder='' class="w290" :disabled="isDisabled"></el-input>
											</el-form-item>
											<el-form-item v-show="isDisabled" label="SSID" prop="ssid">
												<el-input v-model="settingsInfo.ssid" placeholder='' class="w290" ></el-input>
											</el-form-item>
											<el-form-item v-show="isDisabled" label="Encryption" prop="encryption">
												<el-select v-model="settingsInfo.encryption" class="loglanguage" size='mini'>
													<el-option label='psk' value="psk"></el-option>
													<el-option label='psk2' value="psk2"></el-option>
													<el-option label='wpa' value="wpa"></el-option>
													<el-option label='wpa2' value="wpa2"></el-option>
												</el-select>
											</el-form-item>
											<el-form-item v-show="isDisabled" label="Key" prop="key">
												<el-input v-model="settingsInfo.key" placeholder='' class="w290" type='password' :maxlenght=100></el-input>
											</el-form-item>
										</div>
									</div>
								</el-form>
						</div>
						<div class='footer'>	
							<div class="linkbuttonGroup">
								<el-button type="primary" @click="updateSettings"><%=rb.getString("QueDing")%></el-button>
								<el-button @click="closeSettingInfo"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</div>
					</el-dialog>
            </div>
			
        </div>
</div>

<script>
	var faultListVue = new Vue({
		el:'#apInfo',
		data(){
			let serialNumberValidator = (rule,value,cb)=>{
				let reg = /^(?=[^,;()='']+$)[^\$][\x00-\x7f]*/;
				if(value ==''||value ==undefined){
					cb('<%=rb.getString("BuNengWeiKong") %>');
				}else{
					if(!reg.test(value) ){
						cb('<%=rb.getString("apXianZhi") %>');
					}else{
						cb()
					}
				}
				
			}
			let keyValidator =  (rule,value,cb)=>{
				if(value == '' || value == undefined){
					// cb('key<%=rb.getString("BuNengWeiKong") %>,max lenght:100');
					cb()
				}else{
					cb()
				}
			}
			let ssidValidator = (rule,value,cb)=>{
				let reg = /^(?=[^,;()='']+$)[^\$][\x00-\x7f]*/;
				if(value == '' || value == undefined){
					cb()
				}else{
					if(!reg.test(value)){
						cb('<%=rb.getString("apXianZhi") %>');
					}else{
						cb()
					}
				}
				
			}
			return{
				lteTurboEnable: settingVue.selectedRow.lte_turbo_enable,
				tabheight:'100%',
				isDisabled:false,
				modal:true,
				dialogUrl:'',
				sildeHeight:'300px',
				sildeWidth:'490px',
				height:'100%',
				params_ap:{
					enbSerialNumber:window.sessionStorage.getItem('apInfoSn'),
				},
				menus:[],
				rowData:[],
				settingsInfo:{
					serial_number:'',
					ssid:"",
					encryption:'psk',
					key:''
				},
				sourceData:{
					oldSsids:'',
					oldKey:'',
					oldEncryption:''
				},
				pageSize:20,
				isAdd:true,
				showWindowInfo:false,
				settingsRules:{
					serial_number:[
						{validator:serialNumberValidator},
					],
					key:[
						{validator:keyValidator},
					],
					ssid:[
						{validator:ssidValidator},
					]
				},
				datalist:[
					{
						connection_status:'1',
						key:"13",
						ssid:'123',
						serial_number:'123'
					},
					{
						connection_status:'1',
						key:"13",
						ssid:'456',
						serial_number:'456'
					},
				]
			
			}
		},
		methods:{
			init(data){
				var vm = this;
				 
			},
			handerClose(){ //点击页面其他地方菜单收起
		        this.$refs.menuActiveFault.hide();
		    },
			optClick(row,ev){ // 操作项： 1.修改 2.删除
		    	var vm = this,
		    		clearFlag = false;   //清除状态 
				var disabledFlg = false;
				if(row.connection_status == '0' && row.state == '1'){ //  链接状态0 表示离线 和 wifi为1 锁定 不可增删改
					disabledFlg = true
				}
		    	vm.menus= [
						{ label: '<%=rb.getString("XiuGai")%>', cls: "el-icon el-icon-operation-edit", code: 'edit' ,disable:disabledFlg},
						{ label: '<%=rb.getString("ShanChu")%>', cls: "el-icon  el-icon-operation-delete", code: 'del',disable:disabledFlg },
				]
		    	this.rowData = row
		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.$refs.menuActiveFault.show(ev);
					
		    	});
		    },
		    clickMenu(ev){ //操作项点击方法 
		    	var codes = {
		    		edit:this.editInfo,
					del: this.delInfo, //跳转到 信息 函数
		    	}
		    	if(codes[ev.code]){
		    		codes[ev.code](this.$root.rowData)
		    	}
		    },
		    apEnableChange(val) {
		    	var vm = this,
		    		url = '${ctx}/cell/ap/setLTETurbo4Enb.action'
		    		params = {
		    			smallCellCode: settingVue.selectedRow.small_cell_code,
		    			lteTurboEnable: val
		    		};
		    	
		    	axios.post(url,stringify(params)).then(function(res){
		    		var data = res.data;
		    		
		    		if(data['success'] == true) {
		    			vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						});
		    		}else {
		    			vm.lteTurboEnable = val == '1'?'0':'1';
		    			vm.$message({
							message:data.message,
							type:'error',
						});
		    		}
		    	})
		    },
			delInfo(code){　//删除
				var vm = this;
				var url = "${ctx}/cell/ap/modifyAPInfos.action"
				vm.sourceData.oldSsids = code.ssid
				vm.sourceData.oldKey = code.key
				vm.sourceData.oldEncryption = code.encryption
				var rowData = vm.rowData
					valueInfo = 'ssid='+rowData.ssid+';'+'encryption='+rowData.encryption+';'+'key='+rowData.key
					obj={
						serialNumber:rowData.serial_number,
						enbSerialNumber:vm.params_ap.enbSerialNumber,
						sourceSsid:vm.sourceData.oldSsids,
						sourceKey:vm.sourceData.oldKey,
						sourceEbcryption:vm.sourceData.oldEncryption,
						type:'delete',
						section:rowData.section,
						value:valueInfo
					}
			
				vm.$confirm('<%=rb.getString("ShanChuAp")%>','<%=rb.getString("QueRen")%>',{
					customClass:"warningConfirm",
					confirmButtonText:'<%=rb.getString("QueDing")%>', 
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					axios.post(url,stringify(obj)).then(function(res){
						var data = res.data;
						if(data.success){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							vm.$refs["capability_list"].refresh();
						}else{
							vm.$message({
								message:data.message,
								type:'error',
							})
						}
					}).catch(function(error){})
					
				}).catch(()=>{
					
				})
			},
			editInfo(code){ //修改
				var vm = this;
				vm.isAdd = true;
				vm.isDisabled = true;
				vm.showWindowInfo = true
				vm.settingsInfo=vm.rowData;
				vm.sourceData = code //修改赋值
				vm.sourceData.oldSsids = code.ssid
				vm.sourceData.oldKey = code.key
				vm.sourceData.oldEncryption = code.encryption
				setTimeout(function () {
                        initForm(vm.$refs.settingsForm);
                    }, 100)
				
			},

			addInfo(){ //添加
				var vm = this;
				vm.isAdd =false;
				vm.isDisabled = false;
				vm.showWindowInfo = true
				vm.settingsInfo.encryption = '';
				vm.settingsInfo.ssid = '';
				vm.settingsInfo.key = '';
			},
			updateSettings(data){ //修改提交
			var vm = this,obj={},valueInfo='',type='',url='${ctx}/cell/ap/modifyAPInfos.action';
			var tipStr = '<%=rb.getString("WuCanShuBianHua")%>';
				if(!vm.isAdd){
					var rowData = vm.settingsInfo
					valueInfo = 'ssid='+rowData.ssid+';'+'encryption='+rowData.encryption+';'+'key='+rowData.key
					obj={
						serialNumber:rowData.serial_number,
						enbSerialNumber:vm.params_ap.enbSerialNumber,
						sourceSsid:rowData.ssid,
						sourceKey:rowData.key,
						sourceEncryption:rowData.encryption,
						section:'',
						type:'add',
						value:valueInfo
					}
					vm.$refs.settingsForm.validate((r)=>{
						if(r){
							axios.post(url,stringify(obj)).then(function(res){
								var data = res.data;
								if(data.success){
									vm.$message({
										message:'<%=rb.getString("ChengGong")%>',
										type:'success',
									})
								vm.showWindowInfo =false;
								vm.$refs["capability_list"].refresh();
								}else{
									vm.$message({
										message:data.message,
										type:'error',
									})
								}
						
							})
						}
						
					})
					
				}else{
					var rowData = vm.rowData
					valueInfo = 'ssid='+rowData.ssid+';'+'encryption='+rowData.encryption+';'+'key='+rowData.key
					obj={
						serialNumber:rowData.serial_number,
						enbSerialNumber:vm.params_ap.enbSerialNumber,
						sourceSsid:vm.sourceData.oldSsids,
						sourceKey:vm.sourceData.oldKey,
						sourceEncryption:vm.sourceData.oldEncryption,
						type:'update',
						section:rowData.section,
						value:valueInfo
					}
					if(isFormChanged(vm.$refs.settingsForm)){
						vm.$refs.settingsForm.validate((r)=>{
							if(r){
								axios.post(url,stringify(obj)).then(function(res){
								var data = res.data;
								if(data.success){
									vm.$message({
										message:'<%=rb.getString("ChengGong")%>',
										type:'success',
									})
								vm.showWindowInfo =false;
								vm.$refs["capability_list"].refresh();
								}else{
									vm.$message({
										message:data.message,
										type:'error',
									})
								}
						
							})
							}
							
						})
					}else{
						vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
					}
						
					
				}
			},
			closeSettingInfo(){
				this.showWindowInfo =false
			},
			openDialogSuc(){

			},
			
		},
		mounted(){
			this.init()
		}
	});
</script>