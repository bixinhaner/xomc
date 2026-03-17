<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	#ipBlockListPage {
		height: 100%;
		display: flex;
		width: 100%;
		position: relative;
		overflow: hidden;
	}
	#ipBlockListPage .lockItem .el-icon-operation-lock:before {
		color: #F2B354;
	}
	#ipBlockListPage .lockItem .el-icon-status-timeLock:before {
		color: #F2B354;
	}
	#ipBlockListPage .el-icon-common-lock:before {
		font-size: 18px;
	}
</style>

<div id="ipBlockListPage">
	<div class='leftWarp commonWarp'>
		<div class='leftBoxHeader'><%=rb.getString("HeiMingDan")%></div>
		<div class="circleIcon placeholder-bt" style="right:10px;top: 46px" placeholder="<%=rb.getString("XinZeng")%>">		
			<span class="el-icon el-icon-circle-add" @click="addIpBtn"></span>
		</div>
		<el-ctable ref="blockTable" :url='blockListUrl' :query-params="ipParams" id="blockTable" pagination="true" :rownumber=true :row-key="'IP_ADDR'" style="width:100%">
			<template slot="toolbar">
                <div class='commonQuery' style=' border: 0;'>
                	<el-query type="normal" @query="ipQuery" placeholder="<%=rb.getString("IPDiZhi")%>"></el-query>                      
                </div>
	         </template>
	         <!-- IS_LOCK是1表锁定状态，0代表解锁状态 -->
             <el-table-column label='' width="120" prop="">
				<template slot-scope="scope">
					<div>
						<div v-if='scope.row.IS_LOCK == "1" || scope.row.IS_LOCK == "2"' class="el-icon el-icon-operation-unlock" title="<%=rb.getString("JieSuo") %>" @click="unLockInfo(scope.row)" ></div>
						<div v-else class="el-icon el-icon-common-lock" title="<%=rb.getString("YouXiaoQiSuoDing") %>" @click="lockInfo(scope.row)"></div>
						<div class="el-icon el-icon-operation-info" title="<%=rb.getString("XinXi") %>" @click="viewIpInfo(scope.row)" style='padding: 0 6px;'></div>
						<div class="el-icon el-icon-operation-edit" title="<%=rb.getString("XiuGai") %>" @click="editIpInfo(scope.row)" style='padding-right: 6px;'></div>
						<div class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="delIpInfo(scope.row)"></div>
					</div>
				</template>
			</el-table-column>    
	        <el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="IP_ADDR"></el-table-column> 
	        <el-table-column prop='IS_LOCK' label='<%=rb.getString("SuoDingZhuangTai")%>' show-overflow-tooltip="true">
				<template slot-scope='scope'>
					<div v-if='scope.row.IS_LOCK == "0"'>
						<span class='el-icon el-icon-status-unlock'></span><span style='margin-left:5px;'><%=rb.getString("JieSuo")%></span>
					</div>
					<div v-if='scope.row.IS_LOCK == "1" || scope.row.IS_LOCK == "2"' class='lockItem'>
						<span class='el-icon el-icon-operation-lock'></span><span style='margin-left:5px;'><%=rb.getString("YouXiaoQiSuoDing")%></span>
					</div>					
				</template>
			</el-table-column>
	        <el-table-column label="<%=rb.getString("GengXinShiJian")%>" prop="UPD_TIME"></el-table-column>  
	        <el-table-column label="<%=rb.getString("MiaoShu")%>" prop="DESC"></el-table-column>                         
	     </el-ctable>
	</div>
	<!--新建，修改，查看 -->
	<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='ipAddEditViewCard'>
		<div class='rightWarpLayer'>
	 		<div class='rightBoxHeaderHasTip'>
				<div class='headerText'>
					<span>{{ipTitle}}</span>
					<span class='closeIconBox' @click='ipAddEditViewClose'><i class='el-icon el-icon-close'></i></span>
				</div>
			</div>
			<div class='rightWarpLayerContent'>
				<el-form label-position="top" ref="ipForm" :model='ipForm' :rules='ipBlockRules' style='padding: 30px 20px;'>
                    <el-form-item label="<%=rb.getString("IPDiZhi")%>" prop='ip'>
                        <el-input v-model='ipForm.ip' style='width: 320px;' :disabled='ipOperationFlag == "view" || ipOperationFlag == "edit"'></el-input>
                    </el-form-item>
                    <el-form-item label="<%=rb.getString("SuoDingZhuangTai")%>" prop="lock_status">
						<el-select v-model="ipForm.lock_status" size="mini" :disabled='ipOperationFlag == "view"'>
							<el-option label='<%=rb.getString("JieSuo")%>' value='0'></el-option>
							<el-option label='<%=rb.getString("YouXiaoQiSuoDing")%>' value='2'></el-option>
						</el-select>
					</el-form-item>
                    <el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
                        <el-input v-model='ipForm.desc' type='textarea' :rows='3' class="w270" :disabled='ipOperationFlag == "view"'></el-input>
                    </el-form-item>
	            </el-form>
			</div>
			<div class='commonFlex commonBorderTop commonFormFotter' v-if='ipOperationFlag != "view"'>
				<div>
					<el-button type="primary" @click="ipAddEditViewSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="ipAddEditViewClose"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div>
	</div>	
</div>

<script type="text/javascript">
	var ipBlockListPageVue = new Vue({
	    el: '#ipBlockListPage',
	    data() {
	    	var vm = this,
				reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,

	    		validateIp = function(rule,value,callback) {
					if(value == null || value === '' || value.length == 0){
						callback(new Error('<%=rb.getString("IPDiZhiBuNengWeiKong")%>'));
					}else if(!(reg.test(value))){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'));
					}else{
						axios.post('${ctx}/sys/ipBlockList/isExist.action',stringify({
							ip: vm.ipForm.ip
						})).then(function(response){
							var data = response.data;
							if(data["success"]){
								if(data["message"] == true){
								
									callback(new Error('<%=rb.getString("IPYiCunZai")%>'))
								}else{
									callback();
								}
							}else{
								
								callback();
							}
						}).catch(function(error){
							callback()
						})
					}
				},
				validateLockStatus = function(rule,value,callback) {
					if(value == null || value === ''){
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback()
					}
				};
	    	return {
	    		ipParams: {
	    			timeZone: timeZone,
	             	searchText: '',
	            }, 
	            blockListUrl: "${ctx}/sys/ipBlockList/queryIpBlockList.action",
	           	ipAddEditViewCard:false,
	            ipTitle: '<%=rb.getString("TianJia")%>',
	            ipOperationFlag: '',
	            ipForm: {
	            	ip:'',
	            	lock_status: '2',
	            	desc: ''
	           	},	         	          
	           
				rowData:[],
				ipBlockRules: {
					ip: [
                    	{validator: validateIp},
                    ],
                    lock_status: [
                    	{validator: validateLockStatus},
                    ]
	            }
	    	}
	    },
	    methods: {
	    	// 0-解锁， 1-锁定
	    	//锁定-》解锁
	    	unLockInfo(row){
	    		var vm= this;
	    		
	    		vm.$confirm('<%=rb.getString("QueDingJieSuoMa")%>','<%=rb.getString("ShanChu")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/sys/ipBlockList/lockIpBlock.action',stringify({
						ip: row.IP_ADDR,
						lock_status: '0'
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.blockTable.refresh();//表格刷新						
						}else{							
							vm.$message.error(data["message"])
						}
					})
				}).catch(function(){ })	
	    	},
	    	//解锁-》锁定
	    	lockInfo(row){
	    		var vm= this;
	    		
	    		vm.$confirm('<%=rb.getString("QueDingSuoDingMa")%>','<%=rb.getString("ShanChu")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/sys/ipBlockList/lockIpBlock.action',stringify({
						ip: row.IP_ADDR,
						lock_status: '2'
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.blockTable.refresh();//表格刷新						
						}else{							
							vm.$message.error(data["message"])
						}
					})
				}).catch(function(){ })	
	    	},
	    	addIpBtn(){
	    		var vm= this;

	    		vm.ipOperationFlag = 'add';
	    		vm.ipTitle = '<%=rb.getString("TianJia")%>';
	    		vm.$refs.ipForm.resetFields();
	    		vm.ipAddEditViewCard = true;
	    	},
	    	viewIpInfo(row){
				var vm= this;

				vm.ipOperationFlag = 'view';
				vm.ipForm.ip = row.IP_ADDR;
				vm.ipForm.lock_status = row.IS_LOCK == '1' ? '2' : row.IS_LOCK;
				vm.ipForm.desc = row.DESC;
	    		vm.ipTitle = '<%=rb.getString("ChaKan")%>';
	    		vm.ipAddEditViewCard = true;
	    	},
	    	editIpInfo(row){
				var vm= this;

				vm.ipOperationFlag = 'edit';
				vm.ipForm.ip = row.IP_ADDR;
				vm.ipForm.lock_status = row.IS_LOCK == '1' ? '2' : row.IS_LOCK;
				vm.ipForm.desc = row.DESC;
	    		vm.ipTitle = '<%=rb.getString("XiuGai")%>';
	    		vm.ipAddEditViewCard = true;
	    	},
	    	
	    	ipAddEditViewSubmit(){
	    		var vm = this, url = '', params = {};

	    		if(vm.ipOperationFlag == 'add'){
	    			url = '${ctx}/sys/ipBlockList/addIpBlock.action';
	    		}else if(vm.ipOperationFlag == 'edit'){
	    			url = '${ctx}/sys/ipBlockList/updateIpBlockInfo.action';
	    		}
	    		
	    		params.ip = vm.ipForm.ip;
    			params.lock_status = vm.ipForm.lock_status;
    			params.desc = vm.ipForm.desc;
    			params.timeZone = timeZone;
    			
    			saveParams = JSON.stringify(params);
	    		vm.$refs.ipForm.validate((valid) => {
                    if (valid) {
                    	axios.post(url, saveParams, {headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
    						var data = response.data;
    						if(data["success"]){
    							vm.$message.success('<%=rb.getString("ChengGong")%>');
    							vm.$refs.blockTable.refresh();//表格刷新						
    						}else{							
    							vm.$message.error(data["message"])
    						}
    						vm.ipAddEditViewCard = false;
    					})                 	
                    }
                })
	    	},
	    	ipAddEditViewClose(){
				var vm= this;
	    		
	    		vm.ipAddEditViewCard = false;
	    	},	
	    	delIpInfo(row){
				var vm= this;
	    		
				vm.ipAddEditViewCard = false;
	    		vm.$confirm('<%=rb.getString("QueDingYaoShanChuIPMa")%>','<%=rb.getString("ShanChu")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/sys/ipBlockList/deleteIPBlockInfo.action',stringify({
						ip : row.IP_ADDR
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.blockTable.refresh();//表格刷新						
						}else{							
							vm.$message.error(data["message"])
						}
					})
				}).catch(function(){
					
				})	
	    	},
	    	//Ipsec 模块  搜索事件 
	    	ipQuery(val){
	    		Object.assign(this.ipParams,{    			
	    			searchText: val	    		
	    		}) 
	    	},	    	

	    },
	});
	
</script>