<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwAccessControlPage .container .operations{
		right: 10px;
	}
	#egwAccessControlPage .slide-position-top .el-card__body{
		box-sizing: border-box !important;
	}
	#egwAccessControlPage .accessControlMainBox{
		height: 100%;
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	#egwAccessControlPage .container .cmenu{
		z-index: 361!important;
	}
	#egwAccessControlPage .headBtnCls .el-icon::before{
		font-size: 30px;
	}
	#egwAccessControlPage .deviceTable{
		height: 100%;
	}
	#egwAccessControlPage .activeItemCls .el-icon::before{
		font-size: 16px;
		color: #67D972;
	}
	#egwAccessControlPage .noActiveItemCls .el-icon::before{
		font-size: 16px;
		color: #e88282;
	}
	#egwAccessControlPage .enableExplain{
		padding-left: 5px;
		font-size:12px;
		color:#363B4E;
		font-weight: bold;
	}
	#egwAccessControlPage .maskBoxCls{
		position: absolute;
		left: 0;
		right: 0;
		top: 0;
		bottom:0;
		cursor: pointer;
	}
</style>
<div class="pageDefault" id='egwAccessControlPage'>
	<div class="container">
		<div class="accessControlMainBox">
			<!-- 操作按钮 -->
			<div class="operations">
				<div class="placeholder-bt" tip="List">
					<div class="headBtnCls" >
						<span class="el-icon el-icon-circle-List" @click="goListPage"></span>
					</div>
				</div>
			</div>
		
			<div class="deviceTable">
				<el-ctable
					:url="deviceTableUrl"
					:query-params="queryParams" 
					ref="accessControlTable" 
					id="accessControlTable"
					:height="height" 
					@selection-change='deviceSelect'
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div style="display: flex;align-items: center;">
							<h3 style="padding-left: 10px;">Access Status</h3>
							<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("eGWSheBeiBianMa")%> / ECI" style="margin-right: 30px;"></el-query>
							<div style="position:relative;">
								<el-switch v-model="autoEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#CFCFCF" ></el-switch>
								<div class="maskBoxCls"  @click="autoEnableChange"></div>
							</div>
							
							<h3 style="padding-left: 10px;"></h3>
							<span class="enableExplain"><%=rb.getString("WeiShiBieECITiShi")%></span>
						</div>
						
					</template>
						<!-- 列表columns -->
					<el-table-column type="selection" width="45"></el-table-column>
					<el-table-column prop="" width="70">
						<template slot-scope="scope">
							<div style="display:flex;align-items: center;">
								<el-tooltip content='<%=rb.getString("JiaRuDaoBaiMingDan")%>' placement='bottom'>
									<span class="el-icon el-icon-operation-whitelist" @click="putInList(scope.row,'white','single')"></span>
								</el-tooltip>
								<el-tooltip content='<%=rb.getString("JiaRuDaoHeiMingDan")%>' placement='bottom'>
									<span class="el-icon el-icon-operation-blacklist" style="margin-left:10px;" @click="putInList(scope.row,'black','single')"></span>
								</el-tooltip>
							</div>
						</template>
					</el-table-column>
					<el-table-column label='ECI' min-width="120" prop="eci" show-overflow-tooltip></el-table-column>
					<el-table-column prop='status' label='<%=rb.getString("ZhuangTai")%>' min-width="120">
						<template slot-scope="scope">
							<div class="activeItemCls" v-if="scope.row.status == '0'">
								<span class="el-icon el-icon-circle-success"></span>
								<%=rb.getString("YiShiBie")%>
							</div>
							<div class="noActiveItemCls" v-if="scope.row.status == '1'">
								<span class="el-icon el-icon-circle-close"></span>
								<%=rb.getString("WeiShiBie")%>
							</div>
						</template>
					</el-table-column>
					<el-table-column prop='enb_serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
					<el-table-column prop='enb_name' label='eNB Name' min-width="150"></el-table-column>
					<el-table-column prop="egw_serial_number" label="<%=rb.getString("eGWSheBeiBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="gw_ip" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
					<el-table-column prop="gw_port" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
					<el-table-column prop="event_time" label="<%=rb.getString("ShiJian")%>" min-width="150"></el-table-column>
				
				</el-ctable>
			</div>
		</div>
		<!--批量组件弹窗-->
		<el-bulk ref="viewListBulk" target="accessControlTable" :list="selection" row-key="egwSn"  show-prop="egwSn"
			:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
			<template slot="button">
				<a class="linkbutton linkbutton_trend" @click="putInList('','white','batch')"><span><%=rb.getString("JiaRuBaiMingDan")%></span></a>
				<a class="linkbutton linkbutton_trend" @click="putInList('','black','batch')"><span><%=rb.getString("JiaRuHeiMingDan")%></span></a>
			</template>
		</el-bulk>
		<!-- slide -->
		<el-slide ref="egwListSlide" id="egwListSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
<script type="text/javascript">
var egwAccessControlVue = new Vue({
	el:'#egwAccessControlPage',
	data(){
		return {
			deviceTableUrl:"${ctx}/egw/enbCheck/getEgwEnbCheckList.action",
			selection:[],
			
			queryParams:{
				search_text:'',
				like_fields:'egw_serial_number,eci',
				timeZone:timeZone,
			},
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			autoEnable:'true',
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
    },
	watch:{},
	methods:{
		// 初始化
		init(){
			var vm = this;
			axios.post('${ctx}/egw/enbCheck/getAutoSaveBlackSwitch.action').then(function(response){
				var data = response.data;
				vm.autoEnable = data ? 'true' : 'false';
			})
		},
		goListPage(){
			var vm = this;
			vm.slideHeader = false;
			vm.slideUrl = '${ctx}/egw/pageForward/toEgwEnbWhiteBlackListPage.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '';
			
			vm.$refs.egwListSlide.showSlide(()=>{})
		},
		// 设备表格选择事件
		deviceSelect(selection){
			var vm = this;

			vm.selection = selection;
		},
		// 设备表格 模糊查询
		queryDevice(val){
			var vm = this;
			vm.queryParams.search_text= val;
		},
		autoEnableChange(){
			var vm = this,
				url = '${ctx}/egw/enbCheck/updateAutoSaveBlackSwitch.action',
				params = {
					switch:vm.autoEnable == 'true'? 'false' : 'true'
				};
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				var message = '<%=rb.getString("ChengGong")%>';
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
					vm.autoEnable = vm.autoEnable == 'true'? 'false' : 'true';
				}else{
					vm.$message.error(data["message"])
				}
			})
			event.stopPropagation();
		},
		/**
		* 加入黑名单或白名单
		* @param row: 行数据
		* @param listType: 加入名单类型  white：白 black：黑
		* @param opType: 单个/批量  single/batch
		*/
		// 加入黑名单或白名单
		putInList(row,listType,opType){
			var vm = this,
				url = "",
				dataList=[];
			if(opType == "single"){
				dataList.push({eci:row.eci,egw_serial_number:row.egw_serial_number})
			}else{
				vm.selection.map((item)=>{
					dataList.push({eci:item.eci,egw_serial_number:item.egw_serial_number})
				})
			}
			if(listType == "black"){
				var confirmStr = '<%=rb.getString("QueRenJiaRuHeiMingDan")%>';
				url = "${ctx}/egw/enbCheck/saveECIToBlackList.action";
			}else{
				var confirmStr = '<%=rb.getString("QueRenJiaRuBaiMingDan")%>';
				url = "${ctx}/egw/enbCheck/saveECIToWhiteList.action";
			}
			let paramsData = JSON.stringify(dataList);
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				customClass:'warningConfirm',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
							message:message,
							type:'success',
						})
						 vm.$refs.accessControlTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			}).catch(() => {
				
			})
		},
	},
	mounted(){
		this.init();
	}
	
})

</script> 
