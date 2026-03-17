<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
	#userContent .selectInfo{
		 width:100%;
		 height:60px;
		 background:#fff;
		 position:absolute;
		 bottom:0px;
		 box-shadow:0 0 50px rgba(158,200,222,0.35);
	}
	#userContent .selectInfo p{
		font-size:14px;
		color:#363B4E;
		font-weight:700;
		float:left;
		margin-left:20px;
		margin-top:20px;
	}
	#userContent .userTab .cmenu-item:nth-child(5){
		min-height:	1px;
		border-bottom:1px solid #EBEEF5;
		margin-top:5px;
		margin-bottom:5px;
	}
	#userContent .userTab .cmenu-item:nth-child(5):hover{
		background:#fff;
	}
	.el-popover{
		word-break:break-all;
	}
	#userContent .lockItem .el-icon-operation-lock:before{
		color:#F2B354;
	}
	#userContent .lockItem .el-icon-status-timeLock:before{
		color:#F2B354;
	}
	.el-table .cell.el-tooltip{
		min-width: 0px !important
	}

	.right-layer-wrap {
		display: flex;
		flex-direction: column;
		height: 100%;
		width: 350px;
		min-width: 350px;
		overflow: auto;
		margin-left: 15px;
		border-radius: 10px;
		border: 1px solid #E9E9E9;
		background-color: #fff;
	}
	.right-layer-header,
	.right-layer-footer {
		padding: 10px 20px;
	}
	.right-layer-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		border-bottom: 1px solid #E9E9E9;
	}
	.right-layer-header .title {
		margin: 0;
		font-size: 14px;
		font-weight: bold;
		color: rgba(0, 0, 0, 0.8);
	}
	.right-layer-body {
		padding: 20px;
		flex: auto;
		height: 100%;
		overflow: auto;
	}
	.right-layer-footer {
		border-top: 1px solid #E9E9E9;
	}
</style>
<div class="overflow-cls">
<div class='panelDefault' id='userContent' style="min-width: 900px; overflow: hidden;">
	<div v-if="false" class="circleIcon placeholder-bt" v-if='showAddButton' style="top: 40px;" placeholder="<%=rb.getString("TianJia")%>">		
		<span class="el-icon el-icon-circle-add" @click='addItem'></span>
	</div>
	<div style="display: flex;height: 100%;width: 100%;background-color: #F6F7FB;">
		<el-tabs v-model='activeName' style='width:100%;height:100%;flex: auto;overflow: auto;'  @tab-click='clickTab'>
			<el-tab-pane v-if="hasRole" label='<%=rb.getString("JueSe")%>' name='role'>
				<el-ctable ref="roleTable" :id="'roleTable'" :url='roleUrl' :width='width' :height="height"  :query-params="roleParams" pagination="true" @selection-change='selectRole' row-key="id">
					<template slot="toolbar">
						<div class='queryGroup'>
							<el-input v-model='roleForm.role_name' @keyup.enter.native="roleQuery" class='pairgrid-query' placeholder='<%=rb.getString("JueSeMingCheng")%>'></el-input>
							<i @click='roleQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>

						<div v-if="isRoleEditable" class="circleIcon placeholder-bt" style="top: 15px;" placeholder="<%=rb.getString("TianJia")%>">		
							<span class="el-icon el-icon-circle-add" @click='addItem'></span>
						</div>
					</template>
					<el-table-column v-if="batchFlag && isRoleEditable" type='selection' width='55' :selectable='disableRole'></el-table-column>
					<el-table-column label='' width="30" class-name="no-text-tips">
						<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="roleOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
						</template>
					</el-table-column>
					<el-table-column prop='role_name' label='<%=rb.getString("JueSeMingCheng")%>' width='200'></el-table-column>
					<el-table-column prop='batch_operation' label='<%=rb.getString("ZhiChiPiLiang")%>' width="200">
						<template slot-scope="scope">
							<span v-if="scope.row.batch_operation == '1'"><%=rb.getString("Shi")%></span>
							<span v-else><%=rb.getString("Fou")%></span>
						</template>
					</el-table-column>
					<el-table-column prop='user' label='<%=rb.getString("CaoZuoRen")%>' width='200' show-overflow-tooltip="true"></el-table-column>
					<el-table-column prop='update_time' label='<%=rb.getString("GengXinShiJian")%>'></el-table-column>
					<el-table-column prop='desc' label='<%=rb.getString("MiaoShu")%>'></el-table-column>
				</el-ctable>
				<el-cmenu ref='roleMenu' :data='roleMenu' @click='clickRoleMenu'></el-cmenu>
			</el-tab-pane>
			<el-tab-pane v-if="hasGroup" label='<%=rb.getString("YongHuZu")%>' name='group'>
				<el-ctable ref="groupTable" :id="'groupTable'" :url='groupUrl' :width='width' :height="height"  :query-params="groupParams" pagination="true" @selection-change='selectGroup' row-key="group_id">
					<template slot="toolbar">
						<div class='queryGroup'>
							<el-input v-model='groupQueryForm.group_name' @keyup.enter.native="groupQuery" class='pairgrid-query' placeholder='<%=rb.getString("ZuMing")%>'></el-input>
							<i @click='groupQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>

						<div v-if="isGroupEditable" class="circleIcon placeholder-bt" style="top: 15px;" placeholder="<%=rb.getString("TianJia")%>">		
							<span class="el-icon el-icon-circle-add" @click='addItem'></span>
						</div>
					</template>
					<el-table-column v-if="batchFlag && isGroupEditable" type='selection' width='55' :selectable='disableGroup'></el-table-column>
					<el-table-column label='' width="30" class-name="no-text-tips">
						<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="groupOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
						</template>
					</el-table-column>
					<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>' width='200'></el-table-column>
					<el-table-column prop='user_count' label='<%=rb.getString("YongHuShu")%>' width='200' show-overflow-tooltip="true"></el-table-column>
					<el-table-column prop='role_count' label='<%=rb.getString("JueSeShu")%>'></el-table-column>
					<el-table-column prop='upd_user' label='<%=rb.getString("CaoZuoRen")%>'></el-table-column>
					<el-table-column prop='upd_time' label='<%=rb.getString("GengXinShiJian")%>'></el-table-column>
				</el-ctable>
				<el-cmenu ref='groupMenu' :data='groupMenu' @click='clickGroupMenu'></el-cmenu>
			</el-tab-pane>
			<el-tab-pane v-if="cloudFlag == 'false' && hasUser" label='<%=rb.getString("YongHu")%>' name='user' class='userTab'>
				<el-ctable ref="userTable" :id="'userTable'" :url='userUrl' :width='width' :height="height"  :query-params="userParams" pagination="true" @selection-change='selectUser' row-key="id">
						<template slot="toolbar">
							<div class='queryGroup'> 
								<el-input v-model='userQueryForm.searchText' @keyup.enter.native="userQuery" class='pairgrid-query' placeholder='<%=rb.getString("YongHuMingCheng")%>'></el-input>
								<i @click='userQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>

							<div v-if="isUserEditable" class="circleIcon placeholder-bt" style="top: 15px;right: 95px;" placeholder="<%=rb.getString("DaoRu")%>">		
								<span class="el-icon el-icon-circle-import" @click='showImport'></span>
							</div>
							<div v-if="isUserEditable" class="circleIcon placeholder-bt" style="top: 15px;right: 55px;" placeholder="<%=rb.getString("TianJia")%>">		
								<span class="el-icon el-icon-circle-add" @click='addItem'></span>
							</div>
							<div class="circleIcon placeholder-bt" style="top: 15px;" placeholder="<%=rb.getString("DaoChu")%>">		
								<span class="el-icon el-icon-operation-export" @click='exportUser'></span>
							</div>
						</template>
						<el-table-column v-if="batchFlag && isUserEditable" type='selection' width='55' :reserve-selection="true" :selectable='disableUser'></el-table-column>
						<el-table-column label='' width="30" class-name="operationColumn">
							<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" @click="userOptClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
							</template>
						</el-table-column>
						<el-table-column prop='online_status' label='<%=rb.getString("ShouYe_ZaiXianZhuangTai")%>' width='200'>
							<template slot-scope='scope'>
								<div v-if='scope.row.online_status == "Yes"'>
									<span class='statusIcon status_online'><%=rb.getString("ShouYe_ZaiXian")%></span>
								</div>
								<div v-if='scope.row.online_status == "No"'>
									<span class='statusIcon status_offline'><%=rb.getString("ShouYe_BuZaiXian")%></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop='lock_status' label='<%=rb.getString("SuoDingZhuangTai")%>' width='200' show-overflow-tooltip="true">
							<template slot-scope='scope'>
								<div v-if='scope.row.lock_status == 0'>
									<span class='el-icon el-icon-status-unlock'></span><span style='margin-left:5px;'><%=rb.getString("JieSuo")%></span>
								</div>
								<div v-if='scope.row.lock_status == 2' class='lockItem'>
									<span class='el-icon el-icon-operation-lock'></span><span style='margin-left:5px;'><%=rb.getString("YouXiaoQiSuoDing")%></span>
								</div>
								<div v-if='scope.row.lock_status == 1' class='lockItem'>
									<span class='el-icon el-icon-status-timeLock'></span><span style='margin-left:5px;'><%=rb.getString("YouXiaoQiSuoDing")%></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop='user_name' label='<%=rb.getString("YongHuMingCheng")%>'></el-table-column>
						<el-table-column prop='email' label='<%=rb.getString("YouXiang")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("YongHuZu")%>'>
							<template slot-scope='scope'>
								<div v-if='scope.row.group_name == null'></div>
								<div v-else-if='scope.row.group_name.split(",").length == 1'>{{scope.row.group_name}}</div>
								<div v-else>
									<span>{{scope.row.group_name.split(',')[0]}}...</span><el-popover style='word-wrap:break-word' placement='bottom' trigger='click' :content='scope.row.group_name' width='200'><span slot='reference' style='color:#7584FF;cursor:pointer'>[{{scope.row.group_name.split(',').length}}]</span></el-popover>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop='last_login_time' label='<%=rb.getString("ShangCiDengLuShiJian")%>'></el-table-column>
						<el-table-column prop='source' label='<%=rb.getString("LaiYuan")%>'></el-table-column>
				</el-ctable>
				<el-cmenu ref='userMenu' :data='userMenu' @click='clickUserMenu'></el-cmenu>
			</el-tab-pane>
		</el-tabs>

		<!-- Import User-->
		<div v-if="importShow" class="right-layer-wrap">
			<div class="right-layer-header">
				<span class="title"><%=rb.getString("DaoRu")%></span>
				<span @click="closeImport">
					<i class="el-icon el-icon-close"></i>
				</span>
			</div>
			<div class="right-layer-body">
				<el-form ref="importForm" :model="importForm" :rules="importRules" label-position="top">
					<el-form-item label="Excel File" prop="fileName">
						<div style="display: flex;align-items: center;">
							<el-upload style="width: 300px;"
								ref="file"
								:multiple="true" 
								:on-change="fileChange"  
								:show-file-list=false 	                  		
								:action="uploadFileUrl" 
								:data="fileData" 
								:file-list="importForm.fileList" 
								name="uploadFile" 
								accept=".xlsx,.xls"
								:auto-upload="false">
								<el-input :readonly="true" style="width: 100%;" v-model="importForm.fileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
								</el-input>								
								<a slot="trigger" ref="file_up"></a>
							</el-upload>
							<div class='commonFlex' style="margin-left: 10px;">
								<span class='commonNotes12'> (.xls/.xlsx)</span>
							</div>
						</div>
						<div>
							<span style="cursor: pointer;" @click="downloadTpl">
								<i class="el-icon el-icon-common-download"></i>
								<span class="commonNotes12" style="text-decoration: underline;">Export Template</span>
							</span>
						</div>
					</el-form-item>
				</el-form>
			</div>
			<div class="right-layer-footer">
				<el-button type="primary" @click="saveImport"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeImport"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</div>

	<el-slide ref="slide" :subTitle="subTitle" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	 	<template slot='subTitle' v-if='showTitle'>
	 		<span style='color:#4E84FF;margin-left:10px;'>( {{subTitle}} )</span>
	 	</template>
	 </el-slide>
	 <div style='display:flex;width:100%;height:60px;'>
	 	<transition name='el-zoom-in-bottom'>
	 		<div class='selectInfo' v-show='showUser'>
				<p><%=rb.getString("YiXuan")%>&nbsp;(&nbsp;<span style='color:#4E84FF;text-decoration:underline'>{{selectUserNum}}</span>&nbsp;)</p>
				<div style='float:right;margin-top:20px;margin-right:30px;'>
					<el-button-group>
						<el-button type='primary' size='small' style='float:left' @click='logout'><%=rb.getString("QiangZhiTuiChuDengLu")%></el-button>
						<el-dropdown placement='top-end' trigger='click' style='float:left' @command='handleCommand'>
							<el-button type='primary' size='small' style='height:24px;'><%=rb.getString("YouXiaoQiSuoDing")%><i class='el-icon-arrow-down' style='font-size:10px;color:#fff;margin-left:5px;'></i></el-button>
							<el-dropdown-menu slot='dropdown'>
								<el-dropdown-item command='2'><%=rb.getString("YouXiaoQiSuoDing")%></el-dropdown-item>
								<el-dropdown-item command='0'><%=rb.getString("JieSuo")%></el-dropdown-item>
							</el-dropdown-menu>
						</el-dropdown>
						<el-button type='primary' size='small' style='float:left;margin-left:10px;' @click='resetPwdUserInfo'><%=rb.getString("MiMaChongZhi")%></el-button>
					 </el-button-group>
					<el-button-group>
						<el-button type='primary' size='small' @click='moveToGroup'><%=rb.getString("YiDongYongHuZu")%></el-button>
						<el-button type='primary' size='small' @click='delUserInfo'><%=rb.getString("ShanChu")%></el-button>
					</el-button-group>
					<el-button size='small' @click='cancelUserBatch'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
	 	</transition>
	 </div>
	 <el-dialog @closed='closeDialog' title='<%=rb.getString("YiDongYongHuZu")%>' :visible.sync='dialogVisible' width='600px' top='25vh' :close-on-click-modal = false>
	 	<div style='width:560px;height:320px;border:1px solid #DEDFE6'>
	 		<el-ctable :url='moveGroupUrl' ref="moveGroupTable" :id="'moveGroupTable'" :width='width' :height="height"  :query-params="moveGroupParams" @selection-change='selectMoveGroup' pagination="true" row-key="group_id">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input v-model='moveGroupForm.group_name' @keyup.enter.native="queryMoveGroup" class='pairgrid-query' placeholder='<%=rb.getString("ZuMing")%>'></el-input>
						<i @click='queryMoveGroup' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column  type='selection' width='55'></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
			</el-ctable>
	 	</div>
	 	<div style='height:20px;margin-top:5px;'>
	 		<p v-if='showMsg' style='color:#E88282;'><%=rb.getString("QingXuanZeYongYuZu")%></p>
	 	</div>
	 	<el-button-group style='margin-top:20px;'>
			<el-button type='primary' size='small' @click='saveMoveGroup'><%=rb.getString("QueDing")%></el-button>
			<el-button size='small' @click='closeDialog'><%=rb.getString("QuXiao")%></el-button>
		</el-button-group>
	 </el-dialog>
	 <!-- group弹框 -->
	 <div style='display:flex;width:100%;height:60px;'>
	 	<transition name='el-zoom-in-bottom'>
	 		<div class='selectInfo' v-show='showGroup'>
				<p><%=rb.getString("YiXuan")%>&nbsp;(&nbsp;<span style='color:#4E84FF;text-decoration:underline'>{{selectGroupNum}}</span>&nbsp;)</p>
				<div style='float:right;margin-top:20px;margin-right:30px;'>
					<el-button type='primary' size='small' @click='delGroupInfo'><%=rb.getString("ShanChu")%></el-button>
					<el-button size='small' @click='cancelGroupBatch'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
	 	</transition>
	 </div>
	 <!-- role弹框 -->
	 <div style='display:flex;width:100%;height:60px;'>
	 	<transition name='el-zoom-in-bottom'>
	 		<div class='selectInfo' v-show='showRole'>
				<p><%=rb.getString("YiXuan")%>&nbsp;(&nbsp;<span style='color:#4E84FF;text-decoration:underline'>{{selectRoleNum}}</span>&nbsp;)</p>
				<div style='float:right;margin-top:20px;margin-right:30px;'>
					<el-button type='primary' size='small' @click='delRoleInfo'><%=rb.getString("ShanChu")%></el-button>
					<el-button size='small' @click='cancelRoleBatch'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
	 	</transition>
	 </div>
</div>
</div>
<script>
	var userVue = new Vue({
		el:'#userContent',
		data(){
			return{
				batchFlag:batchOperation,
				cloudFlag:isCloudCore,
				activeName: isCloudCore == 'true' ? 'group' : 'user',
				width:'100%',
				height:'100%',
				userUrl:'${ctx}/system/sysuser/queryUserPageList.action',
				userQueryForm:{
					searchText:''
				},
				userParams:{
					searchText:'',
					timeZone:timeZone
				},
				slideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				slideModal:'',
				subTitle:'',
				showAddButton:true,
				userMenu:[],
				userRowData:[],
				selectionUser:[],
				selectUserNum:'',
				showUser:false,
				dialogVisible:false,
				groupUrl:'${ctx}/sys/usergroup/getUserGroupList.action',
				groupQueryForm:{
					group_name:''
				},
				groupParams:{
					group_name:'',
					timeZone:timeZone
				},
				groupMenu:[],
				selectGroupNum:'',
				selectionGroup:[],
				showGroup:false,
				groupRowData:[],
				roleUrl:'${ctx}/sys/role/getRoleSet.action',
				roleParams:{
					timeZone:timeZone,
					operator_code:operator_code,
					role_name:''
				},
				roleForm:{
					role_name:''
				},
				roleMenu:[],
				showRole:false,
				selectRoleNum:'',
				selectionRole:[],
				operType:'',
				showTitle:false,
				roleRowData:[],
				moveGroupUrl:'',
				moveGroupParams:{
					group_name:'',
					timeZone:timeZone
				},
				moveGroupForm:{
					group_name:''
				},
				showMsg:false,
				userCommand:'',

				importForm: {
					fileList: [],
					fileName: '',
				},
				importRules: {
					fileName: [{required: true, message: '<%=rb.getString("QingXianXuanZeWenJian")%>'}]
				},
				fileData: [],
				uploadFileUrl: '',
				importShow: false
			}
		},
		computed: {
			isRoleEditable() {
				return writableMap['CODE_SYSTEM_USERS_ROLE'] == true;
			},
			isGroupEditable() {
				return writableMap['CODE_SYSTEM_USERS_USER_GROUP'] == true;
			},
			isUserEditable() {
				return writableMap['CODE_SYSTEM_USERS_USER'] == true;
			},
			hasRole() {
				return [true, false].includes(writableMap['CODE_SYSTEM_USERS_ROLE']);
			},
			hasGroup() {
				return [true, false].includes(writableMap['CODE_SYSTEM_USERS_USER_GROUP']);
			},
			hasUser() {
				return [true, false].includes(writableMap['CODE_SYSTEM_USERS_USER']);
			}
		},
		methods:{
			//点击tab页签方法
			clickTab(){
				if(this.activeName == 'user'){
					this.$refs.userTable.refresh();
				}
				if(this.activeName == 'group'){
					this.$refs.groupTable.refresh();
				}
				if(this.activeName == 'role'){
					this.$refs.roleTable.refresh();
				}
				this.$refs.userTable && this.$refs.userTable.clearSelection();
				this.$refs.groupTable && this.$refs.groupTable.clearSelection();
				this.$refs.roleTable && this.$refs.roleTable.clearSelection();
			},
			//用户模糊查询
			userQuery(){
				var vm = this;
				Object.assign(vm.userParams, vm.userQueryForm);
			},
			//用户组模糊查询
			groupQuery(){
				var vm = this;
				Object.assign(vm.groupParams, vm.groupQueryForm);
			},
			//角色模糊查询
			roleQuery(){
				var vm = this;
				Object.assign(vm.roleParams, vm.roleForm);
			},
			//移动到用户组模糊查询
			queryMoveGroup(){
				var vm = this;
				Object.assign(vm.moveGroupParams,vm.moveGroupForm);
			},
			//移动到用户组提交方法
			saveMoveGroup(){
				var vm = this;
				var user_ids = vm.$refs.userTable.getChecked();
				var group_ids = vm.$refs.moveGroupTable.getChecked();
				var params = {
						user_ids : user_ids.toString(),
						group_ids : group_ids.toString()
				}
				if(group_ids.length == 0){
					vm.showMsg = true;
					return;
				}else{
					axios.post("${ctx}/system/sysuser/updateUserGroupOfUser.action",stringify(params)).then(function(response){
						var data = response.data;
						var message='<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            userVue.$refs.userTable.clearSelection();
                            userVue.$refs.userTable.refresh();
                            vm.closeDialog();
						}
					})
				}
			},
			/**
			 * 移动到用户组->勾选复选框方法
			 * param selection {Array} 勾选项
			*/
			selectMoveGroup(selection){
				if(selection.length > 0){
					this.showMsg = false
				}else{
					this.showMsg = true
				}
			},
			//取消用户多选
			cancelUserBatch(){
				this.showUser = false;
				this.$refs.userTable.clearSelection();
			},
			//取消角色多选
			cancelRoleBatch(){
				this.showRole = false;
				this.$refs.roleTable.clearSelection();
			},
			//取消用户组多选
			cancelGroupBatch(){
				this.showGroup = false;
				this.$refs.groupTable.clearSelection();
			},
			/**
			 * 勾选用户复选框方法
			 * param selection {Array} 勾选项
			*/
			selectUser(selection){
				this.selectionUser = selection;
				if(selection.length != 0){
					this.showUser = true;
				}else{
					this.showUser = false;
				}
				this.selectUserNum = selection.length;
			},
			//添加按钮跳转页面
			addItem(){
				var vm = this;
				if(vm.activeName == 'user'){
					vm.slideUrl = '${ctx}/system/sysuser/toUserCommonPage.action?type=add',
					vm.slideTitle = '<%=rb.getString("XinJianYongHu")%>';
					vm.slideFooter = true;
					vm.slideHeader = true;
					vm.slidePosition = 'top';
					vm.slideHeight = '100%';
					vm.slideWidth = '100%';
					vm.showAddButton = false;
					vm.showTitle = false;
					vm.operType = 'add';
					vm.$refs.slide.showSlide(function(){
		    	    	vm.slideModal = false
		    	    });
				}else if(vm.activeName == 'group'){
					vm.slideUrl = '${ctx}/sys/usergroup/toGroupCommonPage.action?type=add',
					vm.slideTitle = '<%=rb.getString("XinJianYongHuZu")%>';
					vm.slideFooter = true;
					vm.slideHeader = true;
					vm.slidePosition = 'top';
					vm.slideHeight = '100%';
					vm.slideWidth = '100%';
					vm.showAddButton = false;
					vm.operType = 'add';
					vm.showTitle = false;
					vm.$refs.slide.showSlide(function(){
		    	    	vm.slideModal = false
		    	    });
				}else if(vm.activeName == 'role'){
					vm.slideUrl = '${ctx}/sys/role/toRoleCommonPage.action?type=add',
					vm.slideTitle = '<%=rb.getString("XinJianJueSe")%>';
					vm.slideFooter = true;
					vm.slideHeader = true;
					vm.slidePosition = 'top';
					vm.slideHeight = '100%';
					vm.slideWidth = '100%';
					vm.showAddButton = false;
					vm.operType = 'add';
					vm.showTitle = false;
					vm.$refs.slide.showSlide(function(){
		    	    	vm.slideModal = false
		    	    });
				}
			},
			/**
			 * 禁用用户复选框
			 * param row {object} 行数据  
			 *       index {number} 索引
			*/
			disableUser(row,index){
				//build_in字段代表内置用户
				if(row.build_in == '1'){
					return false;
				}else{
					return true;
				}
			},
			//点击页面其他地方关闭菜单
			handerClose(){
				if(this.activeName == 'user'){
					this.$refs.userMenu.hide();
				}else if(this.activeName == 'group'){
					this.$refs.groupMenu.hide();
				}else if(this.activeName == 'role'){
					this.$refs.roleMenu.hide();
				}
			},
			//点击用户菜单
			userOptClick(row,ev){
				var vm = this;
				var logoutFlag = false;
				var resetPwdFlag = false;
				var lockFlag = false;
				var editFlag = false;
				//内置用户都可以操作
				if(row.build_in == '1'){
					editFlag = true;
					logoutFlag = true;
					resetPwdFlag = true;
					lockFlag = true;
				}
				//只有admin可以操作登出
				if("${is_super_adm}" != '1'){      
					logoutFlag = true;
				}
				//只有admin或运营商管理员可以操作重置密码和解锁用户 
				if("${is_super_adm}" != '1' && "${operator_built_role}" != '1'){  
					// resetPwdFlag = true;
					lockFlag = true;
				}
				// 运营商管理员不可重置自己的密码
				if (row.build_in == "9" && "${operator_built_role}" == "1") {
					resetPwdFlag = true;
				}
				// LDAP用户不可重置自己的密码
				if (row.source == "LDAP") {
					resetPwdFlag = true;
				}
				var lockText = '';
				var lockIcon = '';
				// status是1和2代表锁定状态，0代表解锁状态
				if(row.lock_status == "2" || row.lock_status == '1'){
					lockText = '<%=rb.getString("JieSuo")%>';
					lockIcon = 'el-icon el-icon-operation-unlock';
				}else{
					lockText = '<%=rb.getString("YouXiaoQiSuoDing")%>';
					lockIcon = 'el-icon el-icon-common-lock';
				}
				// role limit
				editFlag = !vm.isUserEditable;

				vm.userRowData = row;
				vm.userMenu = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:"info"},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:"edit",disable:editFlag},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:"del",disable:editFlag},
					{label:'<%=rb.getString("FuZhi")%>',cls:"el-icon el-icon-operation-copy",code:"copy",disable:editFlag},
					{label:'',cls:""},
					{label:lockText,cls:lockIcon,code:"lock",disable: editFlag},
					{label:'<%=rb.getString("MiMaChongZhi")%>',cls:"el-icon el-icon-operation-resetPassword",code:"resetPwd",disable: (resetPwdFlag || editFlag)},
					{label:'<%=rb.getString("QiangZhiTuiChuDengLu")%>',cls:"el-icon el-icon-operation-logout",code:"logout",disable: editFlag}
				]
				vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.userMenu.show(ev);
		    	});
			},
			//点击用户菜单项触发方法
			clickUserMenu(ev){
				if(ev.code == 'info') this.viewUserInfo();
				if(ev.code == 'edit') this.editUserInfo();
				if(ev.code == 'copy') this.copyUserInfo();
				if(ev.code == 'lock') this.lockUserInfo('single');
				if(ev.code == 'del')  this.delUserInfo('single');
				if(ev.code == 'resetPwd') this.resetPwdUserInfo('single');
				if(ev.code == 'logout') this.logout('single');
			},
			//查看用户信息
			viewUserInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/system/sysuser/toUserViewPage.action',
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '800px';
				vm.operType = 'view';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//修改用户信息
			editUserInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/system/sysuser/toUserCommonPage.action?type=modify',
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1220px';
				vm.operType = 'modify';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//复制用户信息
			copyUserInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/system/sysuser/toUserCommonPage.action?type=copy',
				vm.slideTitle = 'Copy';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1220px';
				vm.operType = 'modify';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//点击多选用户弹出框解锁/锁定菜单触发的事件回调
			handleCommand(command){
				this.userCommand = command;
				this.lockUserInfo();
			},
			/**
			 * 锁定/解锁用户
			 * param type {string} 单独操作还是批量操作
			*/
			lockUserInfo(type){
				var vm = this;
				var user_ids = '';
				var lock_status = '';
				// lock_status 0：解锁   2：锁定
				if(type == 'single'){
					user_ids = vm.userRowData.id;
					if(vm.userRowData.lock_status == '0' || vm.userRowData.lock_status == ''){
						lock_status = '2'
					}else{
						lock_status = '0'
					}
					var params = {
							user_ids : user_ids,
							lock_status : lock_status
					}
				}else{
					vm.selectionUser.map(function(item){
						user_ids += item.id + ',';
					})
					user_ids = user_ids.substring(0,user_ids.length-1);
					var params = {
						user_ids : user_ids,
						lock_status : this.userCommand
					}
				}
				axios.post("${ctx}/system/sysuser/lockSystemUser.action",stringify(params)).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
    						message:message,
    						type:'success',
    					})
                        userVue.$refs.userTable.clearSelection();
                        userVue.$refs.userTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			/**
			 * 强制登出用户
			 * param type {string} 单独操作还是批量操作
			*/
			logout(type){
				var vm = this;
				var user_names = '';
				if(type == 'single'){
					user_names = vm.userRowData.user_name;
				}else{
					vm.selectionUser.map(function(item){
						user_names += item.user_name + ','
					})
					user_names = user_names.substring(0,user_names.length-1);
				}
				var confirmStr = '<%=rb.getString("QueRenQiangZhiDengChu")%>'
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:"warningConfirm",
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						axios.post("${ctx}/sys/login/forceLogoutUser.action",stringify({user_names:user_names})).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success',
		    					})
                                userVue.$refs.userTable.clearSelection();
                                userVue.$refs.userTable.refresh();
							}else{
								vm.$message.error(data["message"])
							}
					}).catch(() => {
						
					})
				})
			},
			/**
			 * 用户重置密码
			 * param type {string} 单独操作还是批量操作
			*/
			resetPwdUserInfo(type){
				var vm = this;
				var ids = '';
				var user_names = '';
				if(type == 'single'){
					ids = vm.userRowData.id + "";
					user_names = vm.userRowData.user_name;
					var params = {
							ids : ids,
							user_names : user_names
					}
				}else{
					vm.selectionUser.map(function(item){
						ids += item.id + ',';
						user_names += item.user_name + ','
					})
					var params = {
							ids : ids.substring(0,ids.length-1),
							user_names : user_names.substring(0,user_names.length-1)
					}
				}
				var confirmStr = '<%=rb.getString("QueDingChongZhiMiMa")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/system/sysuser/resetPwd.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            userVue.$refs.userTable.clearSelection();
                            userVue.$refs.userTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			/**
			 * 删除用户信息
			 * param type {string} 单独操作还是批量操作
			*/
			delUserInfo(type){
				var vm = this,
					onlines = vm.selectionUser.map(function(item){
						return item.online_status;
					}),
					user_names = vm.selectionUser.map(function(item){
						return item.user_name;
					}),
					hasOnline = onlines.includes('Yes');
				
				if(type == 'single'){
					var ids = vm.userRowData.id + "";
					hasOnline = vm.userRowData.online_status == 'Yes';
					user_names = vm.userRowData.user_name;
				}else{
					var ids = vm.$refs.userTable.getChecked().toString();
				}
				
				var confirmStr = '<%=rb.getString("QueRenShanChuYongHu")%>';

				var ctner = $('#userContent');
				ctner.addClass('loading');
				
				if(hasOnline) {
					confirmStr = '<%=rb.getString("ShanChuZaiXianYongHuTiShi")%>';
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						// logout users
						axios.post("${ctx}/sys/login/forceLogoutUser.action",stringify({user_names:user_names})).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								axios.post("${ctx}/system/sysuser/deleteUser.action",stringify({ids:ids})).then(function(response){
									var data = response.data;
									var message = '<%=rb.getString("ChengGong")%>';
									if(data["success"]){
										vm.$message({
				    						message:message,
				    						type:'success',
				    					})
                                        userVue.$refs.userTable.clearSelection();
                                        userVue.$refs.userTable.refresh();
									}else{
										vm.$message.error(data["message"])
									}
									ctner.removeClass('loading');
								});
							}else{
								vm.$message.error(data["message"])
							}
						});
					}).catch(() => {
						ctner.removeClass('loading');
					})
				}else {
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						axios.post("${ctx}/system/sysuser/deleteUser.action",stringify({ids:ids})).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success',
		    					})
                                userVue.$refs.userTable.clearSelection();
                                userVue.$refs.userTable.refresh();
							}else{
								vm.$message.error(data["message"])
							}
							ctner.removeClass('loading');
						});
					}).catch(() => {
						ctner.removeClass('loading');
					})
				}
			},
			//关闭划出框
			cancelSlide(){
				var vm = this;
				if(vm.activeName == 'group'){
					if(vm.operType == 'view'){
						vm.$refs.slide.hide();
					}else{
						eventBus.$emit('cancel-group')
					}
				}else if(vm.activeName == 'role'){
					if(vm.operType == 'view'){
						vm.$refs.slide.hide();
					}else{
						eventBus.$emit('cancel-role')
					}
				}else if(vm.activeName == 'user'){
					if(vm.operType == 'view'){
						vm.$refs.slide.hide();
					}else{
						eventBus.$emit('cancel-user')
					}
				}
			},
			//批量操作->移动到用户组
			moveToGroup(){
				this.dialogVisible = true;
				this.moveGroupUrl = '${ctx}/sys/usergroup/getUserGroupList.action';
			},
			//关闭移动到用户组弹框
			closeDialog(){
				this.dialogVisible = false;
				this.$refs.moveGroupTable.clearSelection();
				this.showMsg = false;
			},
			/**
			 * 勾选用户组复选框方法
			 * param selection {Array} 勾选项
			*/
			selectGroup(selection){
				this.selectionGroup = selection;
				if(selection.length != 0){
					this.showGroup = true;
				}else{
					this.showGroup = false;
				}
				this.selectGroupNum = selection.length;
			},
			/**
			 * 禁用用户组复选框
			 * param row {object} 行数据  
			 *       index {number} 索引
			*/
			disableGroup(row,index){
				//built_in 1:内置用户组
				if(row.built_in == '1' || row.built_in == '2'){
					return false;
				}else{
					return true;
				}
			},
			//点击用户组菜单
			groupOptClick(row,ev){
				var vm = this;
				var editFlag = false;
				//built_in 1和2:内置用户组
				if(row.built_in == "1" || row.built_in == "2"){
					editFlag = true;
				}
				// role limit
				editFlag = !vm.isGroupEditable;

				vm.groupRowData = row;
				vm.groupMenu = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:"info"},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:"edit",disable:editFlag},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:"del",disable:editFlag}
				]
				vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.groupMenu.show(ev);
		    	});
			},
			//点击用户组菜单项触发方法
			clickGroupMenu(ev){
				if(ev.code == 'info') this.viewGroupInfo();
				if(ev.code == 'edit') this.editGroupInfo();
				if(ev.code == 'del') this.delGroupInfo('single');
			},
			//查看用户组信息
			viewGroupInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/sys/usergroup/toView.action',
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1130px';
				vm.operType = 'view';
				vm.showTitle = true;
				vm.subTitle = vm.groupRowData.group_name;
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//修改用户组信息
			editGroupInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/sys/usergroup/toGroupCommonPage.action?type=modify',
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1130px';
				vm.operType = 'modify';
				vm.showTitle = false;
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			/**
			 * 删除用户组信息
			 * param type {string} 单独操作还是批量操作
			*/
			delGroupInfo(type){
				var vm = this;
				if(type == 'single'){
					var group_id = vm.groupRowData.group_id + "";
				}else{
					var group_id = vm.$refs.groupTable.getChecked().toString();
				}
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/sys/usergroup/deleteForUI.action",stringify({group_id:group_id})).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            userVue.$refs.groupTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			/**
			 * 勾选角色复选框方法
			 * param selection {Array} 勾选项
			*/
			selectRole(selection){
				this.selectionRole = selection;
				if(selection.length != 0){
					this.showRole = true;
				}else{
					this.showRole = false;
				}
				this.selectRoleNum = selection.length;
			},
			/**
			 * 禁用用户组复选框
			 * param row {object} 行数据  
			 *       index {number} 索引
			*/
			disableRole(row,index){
				//内置角色复选框不能勾选
				if(row.id == '10' || row.id == '11' || row.id == '12' || row.built_in == '8' || row.built_in == '9'){
					return false;
				}else{
					return true;
				}
			},
			//点击角色菜单
			roleOptClick(row,ev){
				var vm = this;
				var editFlag = false;
				var delFlag = false;
				if(row.id == "10" || row.id == '11' || row.id == '12'){
					delFlag = true;
				}
				if(row.built_in == '8' || row.built_in == '9'){
					editFlag = true;
					delFlag = true;
				}

				// role limit
				editFlag = !vm.isRoleEditable;
				delFlag = !vm.isRoleEditable;

				vm.roleRowData = row;
				vm.roleMenu = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:"info"},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:"edit",disable:editFlag},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:"del",disable:delFlag}
				]
				vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.roleMenu.show(ev);
		    	});
			},
			//点击角色菜单项触发方法
			clickRoleMenu(ev){
				if(ev.code == 'info') this.viewRoleInfo();
				if(ev.code == 'edit') this.editRoleInfo();
				if(ev.code == 'del') this.delRoleInfo('single');
			},
			//查看角色信息
			viewRoleInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/sys/role/toRoleCommonPage.action?type=view',
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1130px';
				vm.operType = 'view';
				vm.showTitle = false;
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			//修改角色信息
			editRoleInfo(){
				var vm = this;
				vm.slideUrl = '${ctx}/sys/role/toRoleCommonPage.action?type=modify',
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '1130px';
				vm.operType = 'modify';
				vm.showTitle = false;
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			/**
			 * 删除角色
			 * param type {string} 单独操作还是批量操作
			*/
			delRoleInfo(type){
				var vm = this;
				if(type == 'single'){
					var ids = vm.roleRowData.id + "";
				}else{
					var ids = vm.$refs.roleTable.getChecked().toString();
				}
				var confirmStr = '<%=rb.getString("QueRenShanChuJueSe")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/sys/role/deleteRoleForUI.action",stringify({ids:ids})).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            userVue.$refs.roleTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			//点击划出框保存按钮触发方法
			saveSlide(){
				var vm = this;
				if(vm.activeName == 'group'){
					eventBus.$emit('save-group');
				}else if(vm.activeName == 'role'){
					eventBus.$emit('save-role');
				}else if(vm.activeName == 'user'){
					eventBus.$emit('save-user');
				}
			},
			
			// 导出用户列表
			exportUser() {
				var vm = this,
					exportUrl = '${ctx}/system/sysuser/exportUserInfos.action',
					params = {
						searchText: vm.userQueryForm.searchText,
						timeZone: timeZone
					};

				exportByForm(exportUrl, params);
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file, fileList){ 
				var vm = this,
					fileIndex = file.name.lastIndexOf("."),
					fileType = file.name.substr(fileIndex + 1, file.name.length);
				if(['xls','xlsx'].indexOf(fileType.toLowerCase()) === -1){
					return false;
				}else{
					let arrList = [];
					let uploadFileList = [];
					if(fileList && fileList.length > 0){
						fileList.map((item)=>{
							arrList.push(item.name);
							uploadFileList.push(item.raw)
						})
					}
					vm.importForm.fileName = arrList.join(',');
					vm.importForm.fileList = uploadFileList;
				}
			},
			// 选择文件
			fileSelect(){  
				var vm = this;
				vm.importForm.fileName = '';
				vm.importForm.fileList = [];
				vm.$refs.file.clearFiles();
				vm.$refs['file_up'].click();
			},
			saveImport() {
				var vm = this,
					params = {
						FileName: vm.importForm.fileName,
						uploadFile: vm.importForm.fileList[0],
						timeZone: timeZone
					};
				
				vm.$refs.importForm.validate((valid) => {
					if (valid){
						vm.uploadFileUrl = '${ctx}/system/sysuser/uploadFile.action';
						vm.uploadFiles(vm.uploadFileUrl, params, function(data){
							if (data["success"]) {
								vm.$message.success('<%=rb.getString("ChengGong")%>')
								vm.closeImport();
								vm.$refs.userTable.refresh();
							} else {
								vm.$message.error(data.message)
							}
						})
					} else {
						return false;
					}
				})
			},
			/*
			* 导入函数
			* url：当前的修改或者添加url
			* params： 所有的from参数
			*/
			uploadFiles(url, params, cb) {
				var vm = this,
					xhr = new XMLHttpRequest(),
					fmd = new FormData();
				if (params) {
					for (var key in params) {
						fmd.append(key, params[key]);
					}
				}
				xhr.onreadystatechange = function () {
					if (this.readyState == 4 && xhr.status == 200) {
						var data = JSON.parse(xhr.responseText);
						if (cb && typeof cb == 'function') cb(data);
					}
				}
				xhr.open('post', url);
				xhr.send(fmd);
			},
			downloadTpl() {
				var vm = this,
					url = '${ctx}/system/sysuser/downloadImportUsersTemplate.action';

				exportByForm(url, {});
			},
			showImport() {
				var vm = this;

				vm.importShow = true;
				vm.importForm.fileName = '';
				vm.importForm.fileList = [];
				vm.$nextTick(function(){
					vm.$refs.file.clearFiles();
				});
			},
			closeImport() {
				var vm = this;

				vm.importShow = false;
			}
		}
	})
</script>