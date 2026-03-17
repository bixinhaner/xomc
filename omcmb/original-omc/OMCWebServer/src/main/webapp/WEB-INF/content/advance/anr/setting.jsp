<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#jumpAnrSettingPage .el-form-item__label{
		line-height:22px;
		text-align:left;
	}	
	#jumpAnrSettingPage .el-select .el-input.is-disabled .el-input__inner{
		min-height:26px;
		max-height:26px;
	}
	
	#jumpAnrSettingPage .pairgrid-left,
	#jumpAnrSettingPage .pairgrid-left .el-ctable{
		border:none;
		border-left:none;		
		border-bottom:1px solid #E9E9E9;
	}
	#jumpAnrSettingPage .updateSettingWarp .basicInfoWarp{
		padding:40px 50px 10px;
	}
	#jumpAnrSettingPage .updateSettingWarp .selectDeviceWarp{
		border-top:1px solid #E9E9E9;
	}
	#jumpAnrSettingPage .updateSettingWarp .selectDeviceWarp .selectDeviceInfo{
		padding:30px 50px 20px;
	}
	#jumpAnrSettingPage .updateSettingWarp .basicInfoWarp .newTaskName{
		margin-left:50px;
	}
	#jumpAnrSettingPage .newTaskName .el-input__inner{
		height:28px;
	}
	#jumpAnrSettingPage .newTaskName .el-form-item__label{
		width:120px !important;
	}
	#jumpAnrSettingPage .newTaskName .el-form-item__content{
		margin-left:120px !important;
	}
	#jumpAnrSettingPage .updateSettingWarp .basicInfoWarp .newTaskName .el-input{
		width:348px;
		height:26px;
		line-height:26px;
	}
	#jumpAnrSettingPage .selectDeviceBox{
		margin: 0 52px;
		background:#FFFFFF;
		display:flex;
		height:320px;
	}
	#jumpAnrSettingPage .selectDeviceBox .el-radio{
		margin-right:30px;	
	}
	#jumpAnrSettingPage .leftDeviceSpecific {
		width:260px;
		height:299px;
		border:1px solid #E9E9E9;
	}
	#jumpAnrSettingPage .leftDeviceSpecific p{
		height:36px;
		line-height:36px;
		text-align:center;
		background:#F6F7FB;	
		border-bottom:1px solid #E9E9E9;
	}
	#jumpAnrSettingPage .leftDeviceSpecific .el-radio__label{
		font-size:12px;
	}

	#jumpAnrSettingPage .pairgrid-left .el-query{
		margin-left:0 !important;
	}
	#jumpAnrSettingPage .pairgrid-right{
		top:78px !important;
	}
	#jumpAnrSettingPage .el-card__footer{
		padding:10px 50px;
	}
	#jumpAnrSettingPage .noSelectDevice .el-form-item__error{
		left:50px;
		margin-top:-18px;
	}
	#jumpAnrSettingPage .el-radio{
		font-weight:unset;
	}
	#jumpAnrSettingPage .el-icon-time{
		font-size:14px;
	}
	#jumpAnrSettingPage .searchCon{
		margin-left:18px;
	} 
	#jumpAnrSettingPage .searchCon .el-input--suffix{
		width:460px; 
		height:30px;
	}
	#jumpAnrSettingPage .searchCon .el-input--suffix .el-input__inner{
		height:30px;
		line-height:30px;
		padding-left:15px;
		padding-right:40px;
	}
	#jumpAnrSettingPage .searchCon .el-icon-common-search{
		margin-top:5px;
		margin-right:10px;
	}
	#jumpAnrSettingPage .boxBorderCon .el-radio-group{
		margin-top:6px;
	}		
	#jumpAnrSettingPage .confusedPageTitle{
		background-color: #FFF;		
		align-items: center;
	}	
	#jumpAnrSettingPage .el-form-item{
		margin-bottom:22px;
	}
	#jumpAnrSettingPage .radioFreeMode,
	#jumpAnrSettingPage .executeModeWarp{
		display:flex;
	}
	#jumpAnrSettingPage .anrInterSwitch{
		padding:0 90px;
	}
	#jumpAnrSettingPage .tableWarp{
		border:1px solid #E9E9E9;
		border-left:none;
		border-bottom:none;
		flex:1
	}
	#jumpAnrSettingPage .executeModeWarp .el-form-item__label{
		text-align:right;
	}
</style>
<!-- setting -->
<div id="jumpAnrSettingPage">
	<div class="updateSettingWarp">
		<el-form :model='settingForm' :rules="rules" ref=settingForm :hide-required-asterisk=true>
			<!-- 开关信息 -->
			<div class="basicInfoWarp">
				<el-form-item prop='anrEnable'>
					<div style="display:flex;">
						<label><%=rb.getString("GongNnegKaiGuan")%></label>
						<div class="confusedPageTitle" style="margin-top:4px;margin-left:30px;">					
							<el-switch v-model="settingForm.anrEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#CFCFCF"></el-switch>
						</div>
					</div>					
				</el-form-item>
				<el-form-item prop='anrMode'>
					<div class="confusedPageTitle radioFreeMode">	
						<div style="display:flex;">	
							<label><%=rb.getString("YouHuaMoShi")%></label>				
							<el-radio v-model="settingForm.anrMode" label="0" style="margin-top:7px;margin-left:30px;"><%=rb.getString("ANRZiYouMeShi")%></el-radio>
							<%-- <el-radio style="margin-left:60px;" v-model="settingForm.anrMode" label="1"><%=rb.getString("ANRShouKongMeShi")%></el-radio>
							<div>
								<el-form-item class="newTaskName" prop='anrTimeout' label='<%=rb.getString("ChaoChuShiJian")%>'>
									<el-input v-model="settingForm.anrTimeout" style="width:150px;" :disabled="overTimeDisabled">
										<template slot="append"><%=rb.getString("ANRFenZhong")%></template>
									</el-input>
								</el-form-item>
							</div> --%>
						</div>
					</div>
				</el-form-item>							
				
				<!-- 开关 -->
				<div class="executeModeWarp">
					<el-form-item prop='twoWayEnable'>
						<div style="display:flex;">
							<label><%=rb.getString("ShuangXiangLinQuKaiGuan")%></label>
							<div class="confusedPageTitle" style="margin-top:4px;margin-left:30px;">						
								<el-switch v-model="settingForm.twoWayEnable" active-value="1" inactive-value="0" 
								active-color="#4D84FF" inactive-color="#CFCFCF"></el-switch>
							</div>
						</div>						
					</el-form-item>
					<el-form-item prop='anrInterFreqEnable' class="anrInterSwitch">
						<div style="display:flex;">
							<label><%=rb.getString("YiPinLinQuKaiGuan")%></label>
							<div class="confusedPageTitle" style="margin-top:4px;margin-left:30px;">						
								<el-switch v-model="settingForm.anrInterFreqEnable" active-value="1" inactive-value="0" 
								active-color="#4D84FF" inactive-color="#CFCFCF"></el-switch>
							</div>
						</div>
					</el-form-item>
					<el-form-item prop='anrEutranEnable'>
						<div style="display:flex;">
							<label><%=rb.getString("YiXiTongLinQuKaiGuan")%></label>
							<div class="confusedPageTitle" style="margin-top:4px;margin-left:30px;">						
								<el-switch v-model="settingForm.anrEutranEnable" active-value="1" inactive-value="0" 
								active-color="#4D84FF" inactive-color="#CFCFCF"></el-switch>
							</div>
						</div>	
					</el-form-item> 				
				</div>
			</div>
				
			<!--设备选择  -->
			<div class="selectDeviceWarp">
				<div class="group-title not-extend selectDeviceInfo">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
				</div>
				<div class='selectDeviceBox'>
					<div class="leftDeviceSpecific">
						<p><%=rb.getString("BackupRestoreSheBeiZhiDing")%></p>
						<el-radio-group v-model="deviceType">
							<el-radio style="margin:30px 0px 30px 30px;width:100%" label="all"><%=rb.getString("QuanBu")%></el-radio>
							<el-radio label="specified" style="width:100%;"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
						</el-radio-group>
					</div>
					<div style='width:100%;'>			
						<div style='display:flex;height:300px;width:100%;'>		
							<div class="tableWarp">							
								<el-pairgrid v-if="deviceType == 'specified'"
									:id="'selectDeviceList'" 
									:rownumber="true" 
									ref="backupRestoreTaskTable"  
									@selection-change='selectChange' 
									:right-url="rightUrl"
									:left-url="leftUrl" 
									:height="height" 
									row-key="small_cell_code" 
									:query-params="deviceSelectParams"										
									:title="deviceTitle" 
									:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">	
																	
									<template slot="prev">
										<!-- 设备组选择 -->
										<el-ctable 
										style="width:260px;border-right:1px solid #E9E9E9" 
										:id="'group_list'" 
										:show-pager="false" 
										ref="group"  
										:url="groupUrl" 
										:height="'100%'" 
										:show-header="false" 
										:rownumber="false" 
										:row-key="'id'" 
										@row-click="queryGroupChange" 
										pagination="false">
											<template slot='toolbar'>
												<span style="text-align:center;display:block;"><%=rb.getString("SheBeiZu")%></span>											
											</template>
											<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name" show-overflow-tooltip="true"></el-table-column>
										</el-ctable>
									</template>
									
									<template slot="left">
										<el-table-column type="selection" width="45"></el-table-column>
										<el-table-column prop="connection_status" width="50">
											<template slot-scope="scope">
												<div :class="{
													'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
													'':scope.row.have_connected==2,
													'conn_exc':scope.row.connection_status=='Exception',
													'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
										<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
										<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name"></el-table-column>
									</template>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="searchCon">
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="specifiedDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search" @click="deviceSelectQuery"></i>
											</el-input>
										</div>
					                </template>	
									<!--右侧的下拉表格 -->
									<template slot='right'>
										<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
										<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
									</template>
								</el-pairgrid>
									
								<el-ctable v-if="deviceType == 'all'"
									ref="backupRestoreTaskTableAll" 
									id="selectDeviceListAll" 
									row-key="small_cell_code" 
									:url="deviceUrl" 
									:height="height" 
									:query-params="paramsAll" 
									pagination="true"
									style="border-bottom:1px solid #E9E9E9;height:299px;">
										<el-table-column prop="connection_status" width="50">
											<template slot-scope="scope">
												<div :class="{
													'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
													'':scope.row.have_connected==2,
													'conn_exc':scope.row.connection_status=='Exception',
													'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
										<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
										<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name"></el-table-column>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="searchCon">
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="allDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search"  @click="deviceQueryAll"></i>
											</el-input>
										</div>
					                </template>	
								</el-ctable>
							</div>
						</div>											
					</div>					
				</div>
				<el-form-item prop='selectDevices' class="noSelectDevice">
					<el-input v-model='settingForm.selectDevices' v-show="false"></el-input>
				</el-form-item>
			</div>		
		</el-form>
	</div>
</div>

<script type="text/javascript">
var settingPageConfig = new Vue({
	el:'#jumpAnrSettingPage',
	data(){
		var vm = this;
		var validateCodes = (rule,value,callback) => {
			//功能开关未打开，不校验指定设备勾选（0，关；1，开）
			if(vm.settingForm.anrEnable == '1'){
				if(this.deviceType == 'all'){
					callback()
				}else{
					if(value == '' || value == null){
						callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
					}else{
						callback();
					}
				}
			}else{
				callback()
			}			
		};
		return {
			leftUrl:'',
			rightUrl:'',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			height:'280px',
			width:'90%',
			overTimeDisabled:true,
			settingForm:{
				//timeZone: timeZone,
				anrEnable:'0',
				anrMode:'0',
				//anrTimeout:'',
				selectDevices:'',
				twoWayEnable:'0',
				anrInterFreqEnable:'0',
				anrEutranEnable:'0',
				//earfcnes:'',
				earfcnValue:'',
				earfcnGroup:[],
			},
			earfcnTip:false,
			errorMessage:'',
			paramsAll:{
				//isGnb: 1,
				//timeZone: timeZone,
				search_text:'',
				like_fields: 'serial_number,host_name',
				menu: 'son'
			},
			
			//全部
			deviceSelectParams:{				
				//isGnb: 1,
				group_id:'',
				//timeZone: timeZone,
				search_text:'',
				like_fields: 'serial_number,host_name',
				menu: 'son'
			},
			//运营商
			groupUrl: '${ctx}/system/deviceGroup/getDeviceGroupList.action?isGnb=1',		
			deviceUrl:'',	
			deviceType:'specified',
			selection:[],
			rules:{				
				selectDevices:[
					{validator:validateCodes,trigger:'change'}
				]
			},
			allDevicesSearch:'',
			specifiedDevicesSearch:'',
		}
	},
	watch:{
		deviceType:function(val){
			if(val == 'all'){
				this.specifiedDevicesSearch = '';
				this.deviceSelectParams.search_text = '';
			
			}else{
				this.allDevicesSearch = '';
				this.paramsAll.search_text = '';
				
			}
		},
		selection(){
			var data = this.$refs.backupRestoreTaskTable.getData();
			var selectDevices = '';
			if(data.length != 0){
				data.map(function(item){
					selectDevices += item.small_cell_code + ","
				})
			}
			this.settingForm.selectDevices = selectDevices 
		},

	},
	methods:{ 
		init(){
			var vm = this;			
			vm.$nextTick(function(){
				vm.leftUrl = '${ctx}/cell/cpeinfos/getEnbList.action';	
				vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
			});
			if(sysMain.headType == "gnb"){
				vm.paramsAll.isGnb = 1;
				vm.deviceSelectParams.isGnb = 1;
			}
			//字段回显
			axios.post("${ctx}/anr/getSettings.action").then((res) => {
				var data = res.data;
				//已选设备待后端新增接口	
				if(data){
					vm.settingForm.anrEnable = data.anrEnable;
					vm.settingForm.anrMode = data.anrMode;
					vm.settingForm.twoWayEnable = data.twoWayEnable;
					vm.settingForm.anrInterFreqEnable = data.anrInterFreqEnable;
					vm.settingForm.anrEutranEnable = data.anrEutranEnable;

					if(data.anrEnable === '0' || data.anrEnable === '' || data.anrEnable === null || data.anrEnable === undefined){
						vm.settingForm.anrEnable = '0';
					}else{
						vm.settingForm.anrEnable = '1';
					}
					if(data.anrMode === '0' || data.anrMode === '' || data.anrMode === null || data.anrMode === undefined){
						vm.settingForm.anrMode = '0';
					}else{
						//vm.settingForm.anrMode = '1';
					}
					if(data.twoWayEnable === '0' || data.twoWayEnable === '' || data.twoWayEnable === null || data.twoWayEnable === undefined){
						vm.settingForm.twoWayEnable = '0';
					}else{
						vm.settingForm.twoWayEnable = '1';
					}
					if(data.anrInterFreqEnable === '0' || data.anrInterFreqEnable === '' || data.anrInterFreqEnable === null || data.anrInterFreqEnable === undefined){
						vm.settingForm.anrInterFreqEnable = '0';
					}else{
						vm.settingForm.anrInterFreqEnable = '1';
					}
					if(data.anrEutranEnable === '0' || data.anrEutranEnable === '' || data.anrEutranEnable === null || data.anrEutranEnable === undefined){
						vm.settingForm.anrEutranEnable = '0';
					}else{
						vm.settingForm.anrEutranEnable = '1';
					}
					
					
					if(data.selectAll == 'true'){
						vm.deviceType = 'all'
					}else{
						vm.deviceType == 'specified';
						vm.rightUrl = '${ctx}/anr/getTaskSelectedList.action';
					}					
					initForm(vm.$refs.settingForm); 
				}				
			});						
		},

		
		/**
		*  设备列表里的设备组选项变化
		* @param row{object}: 表格行数据
		**/
		queryGroupChange(row){ 
			if(row) {
				this.deviceSelectParams.group_id = row.id;
			}
		},
		
		/**
		 * 全选操作
		 * @param selection:选择的数据
		*/
		selectChange(selection){
			this.selection = selection;
		}, 
		
		//表格搜索
		deviceSelectQuery(val){
			this.deviceSelectParams.search_text = this.specifiedDevicesSearch;
		},
		
		//表格搜索
		deviceQueryAll(val){
			this.paramsAll.search_text = this.allDevicesSearch;
		},
				
		//保存
		taskSubmit(){
	    	var vm = this;
	    	var params = {};

	    	vm.$refs.settingForm.validate((valid) => {
	    		if(valid){
	    			if(vm.deviceType == 'all'){
	    				params.selectAll = "true";
	    				params.selectDevices = "";
	    			}else{
	    				params.selectDevices = vm.settingForm.selectDevices;
	    			}
	    			  
	    			params.anrEnable = vm.settingForm.anrEnable;
	    			params.anrMode = vm.settingForm.anrMode;
	    			params.twoWayEnable = vm.settingForm.twoWayEnable;
	    			params.anrInterFreqEnable = vm.settingForm.anrInterFreqEnable;
	    			params.anrEutranEnable = vm.settingForm.anrEutranEnable;	    				
	   			
	    	    	axios.post('${ctx}/anr/updateSettings.action', stringify(params)).then(function(response){
   						var data = response.data;
   						if (data["success"]) {
   							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-settingPage')
   						} else {   							
   							vm.$message.error(data["message"]);
   						}
   					}).catch(function(error){})
	    		}
	    	})
		},		
	
		// 关闭新建弹窗
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(isFormChanged(vm.$refs.settingForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					anrSettingVue.$refs.slide.hide();
				}).catch(() => {
					
				})
			}else{
				anrSettingVue.$refs.slide.hide();
			}
		},
	},

	mounted(){
		this.init();
		eventBus.$off('taskSave-ok').$on('taskSave-ok',this.taskSubmit);
		eventBus.$off('hander-cancel').$on('hander-cancel',this.cancel);
	}
})
</script>