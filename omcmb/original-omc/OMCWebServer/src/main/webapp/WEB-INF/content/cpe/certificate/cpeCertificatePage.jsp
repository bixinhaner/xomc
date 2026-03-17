<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
	#cpeCertificate{
		height: 100%;
		display: flex;
		width:100%;
	}
	#cpeCertificate .el-tabs {
		width: 100%;
	}	
	ssueStatus{
		margin-top:4px;
		font-size:17px; 
		margin-right:10px;
		float:left;
	}
	.issueNoColor:before{		
		color:#CFCFCF;
	}
	.issueOkColor:before{
		color:#67D972;
	}
	.issueErrorColor:before,
	.statusError:before{
		color:#E88282;
	}
	.el-bulk .el-icon-close{
		font-size:16px;
		top:14px;
	}
	.activeStatusItem .el-icon,.inactiveStatusItem .el-icon, .statusItemInProgress .el-icon, .statusItemSuccess .el-icon, .statusItemFail .el-icon{
		font-size:20px;
		vertical-align:bottom;
		margin-right:5px;
	}
	.activeStatusItem .el-icon::before, .statusItemSuccess .el-icon::before{
		color:#67D972;
	}
	.inactiveStatusItem .el-icon::before,.statusItemFail .el-icon::before{
		color:#E88282;
	}
	.flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	.flex-form .el-form-item {
		margin-right: 100px;
		margin-bottom: 10px!important;
	}
	#cpeCertificate .certTableQueryCls .transition-box{
		width:100%;
	}
	#cpeCertificate .certTableQueryCls .transition-box .el-input--mini{
		margin-top: 7px;
	}
	.settingFormCls .el-form-item{
		margin-bottom: 10px;
		margin-left: 20px;
	}
	.settingFormCls .itemCls{
		margin-top: 6px;
	}
	.settingFormCls .validTimeWarnCls{
		font-size: 12px;
		color: #BBBBBB;
		display: inline-block;
		line-height: 18px;
		position: relative;
		top: 0px;
	}
	.validTimeWarnCls .el-icon:before{
		color: #BBBBBB;
		font-size: 12px;
	}
	.settingFormCls .validTimeUnitBox{
		display: inline-block;
		height: 26px;
		width: 36px;
		line-height: 26px;
		color: #666666;
		text-align: center;
		background-color: #F5F7FA;
		border:1px solid #DCDFE6;
		box-sizing: border-box;
		margin-left: -4px;
		position: relative;
		top:1px;
	}
	.settingFormCls .el-form-item__error{
		padding-top: 0px;
		margin-top: -5px;
	}
	.automaticEnableClickBox{
		height: 20px;
		width: 40px;
		background-color: red;
		position: absolute;
		top:8px;
		left: 20px;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	#cpeCertificate .certImport{
		right: 20px;
		top:43px;
	}
</style>
<!--CPE证书页面  -->
<div id="cpeCertificate">
	<!--操作按钮 -->
	<div v-if="hasCertRole" class="circleIcon placeholder-bt certImport" placeholder="<%=rb.getString("SheZhi")%>">		
		<span class="el-icon el-icon-circle-setting" @click="certAutoUpdateSettings"></span>
	</div>
	<!--主体内容-->
	 <el-tabs v-model="activeName">
	 	<el-tab-pane label="<%=rb.getString("ZhengShu")%>" name="certificateActive"><!--:url="cpeCertUrl" :data="certData" -->
	 		<el-ctable ref="certTable" :time="6"  :url="cpeCertUrl" :query-params="queryParams" id="certTable" :limit="limitBatch"
            	:page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='batchSelect'>
                <template slot="toolbar">
                    <div class="certTableQueryCls">
                        <el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery'
							placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("CPEMacAddress")%> / <%=rb.getString("IMSI")%>"
							:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
							<template slot="form">
								<el-form :model='queryForm' ref="queryForm" label-position="left" class="flex-form" label-width="140px">
									<el-form-item label='<%=rb.getString("CPEBianMa")%>' prop='serialNumber'>
										<el-input v-model='queryForm.serialNumber' size="mini"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("CPEName")%>' prop='cpeName'>
										<el-input v-model='queryForm.cpeName' size="mini"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("CPEMacAddress")%>' prop='macAddress'>
										<el-input v-model='queryForm.macAddress' size="mini"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("IMSI")%>' prop='imsi'>
										<el-input v-model='queryForm.imsi' size="mini"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ZhengShuMingCheng")%>' prop='certName'>
										<el-input v-model='queryForm.certName' size="mini"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ZhengShuZhuangTai")%>' prop='certStatus'>
										<el-select v-model="queryForm.certStatus" size="mini">
											<el-option v-for="item in certStatusList" :label="item.text" :value="item.value"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GengXinJieGuo")%>' prop='updateStatus'>
										<el-select v-model="queryForm.updateStatus" size="mini">
											<el-option v-for="item in updateStatusList" :label="item.text" :value="item.value">
											</el-option>
										</el-select>
									</el-form-item>
								</el-form>
							</template>
						</el-query>                      
                    </div>
                </template>
                
                <el-table-column v-if="hasCertRole" type="selection" :reserve-selection="true"></el-table-column>
                <el-table-column v-if="hasCertRole" width="70">
                    <template slot-scope="scope">
						<div style="display:flex;align-items: center;">
							<el-tooltip content='<%=rb.getString("ShouDongChuFaGengXinZhengShu")%>' placement='bottom'>
								<span class="el-icon el-icon-operation-CertUpdate disabled" v-if="scope.row.certAutoUpdateEnable != '1'"></span>
								<span class="el-icon el-icon-operation-CertUpdate" v-if="scope.row.certAutoUpdateEnable == '1'" @click="certUpdate(scope.row,event)"></span>
							</el-tooltip>
							<el-tooltip content='<%=rb.getString("TongBuZhengShuJiChuXinXi")%>' placement='bottom'>
								<span class="el-icon el-icon-operation-synchronize" style="margin-left:10px;" @click="syncCertInfo(scope.row,event)"></span>
							</el-tooltip>
						</div>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ZhengShuZiDongGengXinKaiGuan")%>" prop="certAutoUpdateEnable" width="180"  show-overflow-tooltip sortable style="position:relative;">
					<template slot-scope="scope">
						<div v-if="scope.row.certAutoUpdateEnable == '1' || scope.row.certAutoUpdateEnable == '0'" >
							<div v-if="hasCertRole" class="automaticEnableClickBox" @click="automaticEnableChange(scope.row)"></div>
							<el-switch v-model="scope.row.certAutoUpdateEnable" style="height: 18px;margin-left: 10px;"
								:disabled="!hasCertRole"
								active-color="#4D84FF"
								active-value="1"
								inactive-value="0">	
							</el-switch>
							<span v-if="scope.row.certAutoUpdateEnable == '1'">Enable</span>
							<span v-if="scope.row.certAutoUpdateEnable == '0'">Disable</span>
						</div>
						<div style="margin-left:55px;" v-else> -- </div>
					</template>
				</el-table-column>   
                <el-table-column label="<%=rb.getString("CPEBianMa")%>" prop="serialNumber" show-overflow-tooltip min-width="160"></el-table-column>   
				<el-table-column label="<%=rb.getString("CPEName")%>" prop="cpeName" show-overflow-tooltip  min-width="100"></el-table-column>
				<el-table-column label="<%=rb.getString("CPEMacAddress")%>" prop="macAddress" min-width="160" show-overflow-tooltip></el-table-column>
				<el-table-column label="<%=rb.getString("IMSI")%>" prop="imsi" min-width="140" show-overflow-tooltip></el-table-column>
				<el-table-column label="<%=rb.getString("ZhengShuMingCheng")%>" prop="certName" min-width="160" show-overflow-tooltip></el-table-column>
                <el-table-column label="<%=rb.getString("ZhengShuZhuangTai")%>" prop="certStatus" min-width="160" show-overflow-tooltip sortable>
                	<!-- 证书当前状态 : 0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 -->	
                	<template slot-scope="scope">				
						<div v-if="scope.row.certStatus == '0'">	
							<span class="el-icon el-icon-status-invalidCert issueStatus issueErrorColor" style="margin-right:5px;"></span><%=rb.getString("WuXiaoDe")%>
						</div>
						<div v-if="scope.row.certStatus == '1'">
							<span class="el-icon el-icon-status-validCert issueStatus issueOkColor" style="margin-right:5px;"></span><%=rb.getString("YouXiaoDe")%>
						</div>
					</template>
                </el-table-column>  
				<el-table-column label="<%=rb.getString("YouXiaoKaiShiShiJian")%>" prop="certValidStartTime" min-width="160" show-overflow-tooltip></el-table-column>
				<el-table-column label="<%=rb.getString("YouXiaoJieShuShiJian")%>" prop="certValidEndTime" min-width="160" show-overflow-tooltip></el-table-column>
                <el-table-column label="<%=rb.getString("GengXinJieGuo")%>" min-width="140" prop="certUpdateStatus" sortable>
					<!-- 证书更新结果 : 0 == 失败，1 == 成功  -->	
                	<template slot-scope="scope">				
						<div v-if="scope.row.certUpdateStatus == '0'">	
							<%=rb.getString("ShiBai")%>
						</div>
						<div v-if="scope.row.certUpdateStatus == '1'">
							<%=rb.getString("ChengGong")%>
						</div>
					</template>
				</el-table-column>     
				<el-table-column label="<%=rb.getString("ShangCiGengXinShiJian")%>" prop="certUpdateTime" min-width="160" show-overflow-tooltip sortable></el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="certUpdateFailureReason" min-width="160" show-overflow-tooltip></el-table-column>
            </el-ctable>           
	 	</el-tab-pane>
	 </el-tabs>
	<!--自动更新 配置弹窗-->
	<el-dialog :title="'<%=rb.getString("ZhengShuZiDongGengXinSheZhi")%>'" :visible.sync="showSettingDialog" width="500" 
		:close-on-click-modal="false" top="30vh" @close="clearSettingDialog" :append-to-body="true">
		<el-form  :model="settingForm" ref="settingForm" :rules='settingRules' label-position="left" label-width="140px" class="settingFormCls">
			<el-form-item label="<%=rb.getString("IPDiZhi")%>" prop="ipAddress">
				<div class="itemCls">
					<el-input v-model="settingForm.ipAddress" style="width:150px;"></el-input>
					<div class="validTimeWarnCls"><span class="el-icon-circle-info el-icon" style="margin-right:5px;"></span>192.168.14.20</div>
				</div>
			</el-form-item>
			<el-form-item   label="">
				<div class="validTimeWarnCls"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("AnQuanShiJianTiShi")%></div>
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="settingSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="clearSettingDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--自动更新开关打开 弹窗-->
	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="showOpenCertAutoUpdateEnableDialog" width="500" 
		:close-on-click-modal="false" top="30vh" @close="showOpenCertAutoUpdateEnableDialog = false">
		<div style="margin-bottom:10px;"><%=rb.getString("QueDingDaKaiZiDongGengXin")%></div>
		<div style="font-size:12px;color:#B4B4B4"><%=rb.getString("IPDiZhi")%>:{{currentIpAddress}}</div>
		<div style="font-size:12px;color:#B4B4B4"><%=rb.getString("AnQuanShiJian")%>:<%=rb.getString("AnQuanShiJianTiShi")%></div>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="submitOpenCertAutoUpdateEnable"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="showOpenCertAutoUpdateEnableDialog = false"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--批量组件弹窗-->
	<el-bulk ref="cpeCertBulk" target="certTable" :list="selectionData" row-key="cpeCode" show-prop="serialNumber" class="el-bulk"
		:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
		<template slot="button">	
			<a class="linkbutton" @click="batchCertAutoUpdateEnable('1')"><span><%=rb.getString("KaiQiZiDongGeng")%></span></a>
			<a class="linkbutton linkbutton_nowanna" @click="batchCertAutoUpdateEnable('0')" style="margin-left:-14px;"><span><%=rb.getString("GuanBiZiDongGeng")%></span></a>		
			<a class="linkbutton" @click="batchCertUpdate"><span><%=rb.getString("GengXin")%></span></a>
		</template>
	</el-bulk>
</div>

<script type="text/javascript">
	var cpeCertVue = new Vue({
	    el: '#cpeCertificate',
	    data() {
			var vm = this;
			var validatorIpAddress = (rule,value,callback) => {
					var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
					
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("IPDiZhiBuNengWeiKong")%>'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPDiZhiFeiFa")%>'))
						}
					}
				},
				validateValidTime = (rule,value,callback) => {
					var reg = /^([1-9]\d?|100)$/;
					
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%>:1-100'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%>:1-100'))
						}
					}
				};
	    	return {	    		 
	    		activeName: 'certificateActive',
	    		menus: [],
	    		pageSize:50,
	    		queryParams: {
	    			timeZone: timeZone,
	             	searchText: '',
					serialNumber :'', 
					cpeName:'',
					macAddress:'',
					imsi:'',
					certName:'',
					certStatus:'',
					updateStatus:''
	            }, 
				queryForm:{
					serialNumber :'', 
					cpeName:'',
					macAddress:'',
					imsi:'',
					certName:'',
					certStatus:'',
					updateStatus:''
				},
	            cpeCertUrl:"${ctx}/cell/cpe/cert/queryCPECertList.action",
	            selectionData:[],
	            rowData:[],
				certStatusList:[
					{value:'',text:'<%=rb.getString("QuanBu")%>'},
					{value:'0',text:'<%=rb.getString("WuXiaoDe")%>'},
					{value:'1',text:'<%=rb.getString("YouXiaoDe")%>'}
				],
				updateStatusList:[
					{value:'',text:'<%=rb.getString("QuanBu")%>'},					
					{value:'0',text:'<%=rb.getString("ShiBai")%>'},
					{value:'1',text:'<%=rb.getString("ChengGong")%>'}
				],
				showSettingDialog:false,
				settingForm:{
					ipAddress:'',
				},
				settingRules:{
					ipAddress:[
						{validator:validatorIpAddress,trigger:'blur'}
					],
				},
				showOpenCertAutoUpdateEnableDialog:false,
				currentIpAddress:'',
				operationType:'' // 操作类型 单个 single    批量 batch 
	    	}
	    },
	    watch: {
	    	
		},
	    methods: {
			
			// 模糊查询
	    	query(val){
				var vm = this;
				vm.resetQuery();
				vm.queryParams.searchText = val;
			},
			// 高级查询
			advanceQuery(){
				var vm = this;
				Object.assign(vm.queryParams, vm.queryForm);
			},
			// 查询重置
			resetQuery(){
				var vm = this,
					params = {
						searchText: '',
						serialNumber :'', 
						cpeName:'',
						macAddress:'',
						imsi:'',
						certStatus:'',
						updateStatus:''
					};
			
				Object.assign(vm.queryForm, params);
				Object.assign(vm.queryParams, params);
			},
			// 自动更新按钮点击事件	    
			automaticEnableChange(row){
				var vm = this;

				vm.rowData = row;
				if(row.certAutoUpdateEnable == '1'){
					var url="${ctx}/cell/cpe/cert/autoUpdate.action",
						confirmMsg = '<%=rb.getString("QueDingGuanBiZiDongGengXin")%>',
						params={
							cpeCodes:row.cpeCode,
							enableAutoUpdate:'0',
							ipAddress:row.ipAddress,
							validTime:'7'
						};

					vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						closeOnClickModal:false
					}).then(() => {
						axios.post(url,stringify(params)).then(function(response){
							let data = response.data
							if(data.success){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.$refs.certTable.refresh();
							}else{
								vm.$message({
									message: data.message,
									type:'error',
								});
							}
						}).catch(function(error){
							
						})
	    			}).catch()
				}else{
					vm.operationType = 'single';
					
					axios.post("${ctx}/cell/cpe/cert/getConfigInfo.action").then(function(response){
						let data = response.data
						vm.currentIpAddress = data.ipAddress;
						vm.showOpenCertAutoUpdateEnableDialog = true;
						// if(vm.currentIpAddress){
						// 	vm.showOpenCertAutoUpdateEnableDialog = true;
						// }else{
						// 	vm.$message({
						// 		message: '<%=rb.getString("QingxianPeiZhiIpDiZhiTiShi")%>',
						// 		type:'warning',
						// 	});
						// }
					}).catch(function(error){
						
					})
				}
				event.stopPropagation();
			},
	    	/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			batchSelect(selection){
				var vm = this; 
			    vm.selectionData = selection; 
			},
			// 证书更新
			certUpdate(row){
				var vm = this,
					url ='${ctx}/cell/cpe/cert/update.action',
					confirmMsg = '<%=rb.getString("QueDingGengXinZhengShu")%>',
					params = {
						cpeCodes:row.cpeCode,
					};	    		
				vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.certTable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
			// 同步证书基础信息
			syncCertInfo(row){
				var vm = this,
					url ='${ctx}/cell/cpe/cert/refresh.action',
					params = {
						cpeCodes:row.cpeCode,
					};
				axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.certTable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){})
			},
			// 批量开启 关闭自动更新
			batchCertAutoUpdateEnable(status){
				var vm = this;
					
				if(status === '1'){
					vm.operationType = 'batch';
					axios.post("${ctx}/cell/cpe/cert/getConfigInfo.action").then(function(response){
						let data = response.data
						vm.currentIpAddress = data.ipAddress;
						if(vm.currentIpAddress){
							vm.showOpenCertAutoUpdateEnableDialog = true;
						}else{
							vm.$message({
								message: '<%=rb.getString("QingxianPeiZhiIpDiZhiTiShi")%>',
								type:'warning',
							});
						}
					}).catch(function(error){
						
					})
				}else{
					var codeList=[],
						confirmMsg='',
						url='${ctx}/cell/cpe/cert/autoUpdate.action',
						params = {
							cpeCodes:''
						};
					vm.selectionData.map(function(item,idx){
						codeList.push(item.cpeCode);
					});
					params.cpeCodes = codeList.join(',');
					confirmMsg = '<%=rb.getString("QueDingGuanBiZiDongGengXin")%>'

					vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						closeOnClickModal:false
					}).then(() => {
						axios.post(url,stringify(params)).then(function(response){
							let data = response.data
							if(data.success){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.$refs.certTable.clearSelection();
								vm.$refs.certTable.refresh();
							}else{
								vm.$message({
									message: data.message,
									type:'error',
								});
							}
						}).catch(function(error){
							
						})
	    			}).catch()
				}
	    	
			},
			batchCertUpdate(){
				var vm = this,
					url ='${ctx}/cell/cpe/cert/update.action',
					confirmMsg = '<%=rb.getString("QueDingGengXinZhengShu")%>',
					cpeCodeList=[],
					params = {
						cpeCodes:'',
					};	  
				vm.selectionData.map((item)=>{
					cpeCodeList.push(item.cpeCode)
				})
				params.cpeCodes = cpeCodeList.join(',');
				vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.certTable.clearSelection();
							vm.$refs.certTable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
			// 打开 设置弹窗
			certAutoUpdateSettings(){
				var vm = this;

				axios.post("${ctx}/cell/cpe/cert/getConfigInfo.action").then(function(response){
					let data = response.data
					vm.settingForm.ipAddress = data.ipAddress;
					vm.showSettingDialog = true;
				}).catch(function(error){
					
				})
				
			},
			// 设置提交
			settingSubmit(){
				var vm = this,
					url="${ctx}/cell/cpe/cert/updateConfigInfo.action",
					params={
						ipAddress:vm.settingForm.ipAddress
					};
				vm.$refs.settingForm.validate((valid) => {
					if(valid){
						axios.post(url,stringify(params)).then(function(response){
							let data = response.data
							if(data.success){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success',
								});
								vm.$refs.certTable.refresh();
								vm.showSettingDialog = false;
							}else{
								vm.$message({
									message: data.message,
									type:'error',
								});
							}
						}).catch(function(error){
							
						})
					}else{
						return false;
					}
				})
				
			},
			// 设置弹窗关闭
			clearSettingDialog(){
				var vm = this;
				vm.showSettingDialog = false;
				vm.settingForm.ipAddress = '';
			},
			// 单个 、 批量自动更新开关 打开事件
			submitOpenCertAutoUpdateEnable(){
				var vm = this,
					cpeCodeList=[],
					url='${ctx}/cell/cpe/cert/autoUpdate.action',
					params = {
						cpeCodes:'',
						enableAutoUpdate:'1',
						ipAddress:'',
						validTime:'7'
					};
				if(vm.operationType == 'single'){
					params.cpeCodes = vm.rowData.cpeCode;
				}else{
					vm.selectionData.map(function(item,idx){
						cpeCodeList.push(item.cpeCode);
					});
					params.cpeCodes = cpeCodeList.join(',');
				}
				axios.post("${ctx}/cell/cpe/cert/getConfigInfo.action").then(function(response){
					let data = response.data
					params.ipAddress = data.ipAddress;
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							if(vm.operationType == 'batch'){
								vm.$refs.certTable.clearSelection();
							}
							vm.$refs.certTable.refresh();
							vm.showOpenCertAutoUpdateEnableDialog = false;
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){
						
					})
				}).catch(function(error){
					
				})
				
			
			}
	    
	    },
		computed: { 
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			hasCertRole() {
				return writableMap.CODE_CPE_IPSEC_CERT == true;
			}
	    },
	    mounted(){
			
		}
	});
	

</script>