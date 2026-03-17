<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.selected-tmp-con{
		margin:22px 10px 0px 12px;
		height:48px;
		border:1px solid #D1ECF5;
		border-radius:5px;
		display:flex;
		flex:1 1 auto;
		padding:0px 14px;
		border-box:box-sizing;
		align-items:center;
		justify-content:space-between;
	}
	.select-box-container{
		display:flex;
		flex-direction:row;
	}
	.select-box{
		width:140px;
		height:28px;
		border-radius:2px;
		border:1px solid #EEF0F1;
		background:#F7F7F7;
		display:flex;
		justify-content:space-between;
		align-items:center;
		padding:0px 10px;
		margin-right:10px;
	
	}
	.select-box span{
		overflow:hidden;
		white-space:nowrap;
		text-overflow:ellipsis;
	}
	.search-container{
		display:flex;
		flex-direction:row;
	}
	.tool-container{
		margin-left:20px;
		border:1px solid #E3E3E3;
		border-radius:2px;
		display:flex;
		height:30px;
		align-items:center;
		align-self:flex-end;
	}

	.tool-container .icon-box-div{
		border-left:1px solid #E9E9E9;
		display:inline-block;
		min-width:30px;
		height:100%;
		display:flex;
		padding:0px 5px;
		align-items:center;
	}
	#alarmCardTempList .has-gutter .cell .el-checkbox .el-checkbox__input{
		display:none;
	}
	.tableCard .el-card__footer{
		height:0px;
		border:none;
	}
</style>

<div class="" id="card_alarm_app">
	<div class="buttonGroup placeholder-bt" style='right:45px;' placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click='closeViewTemple'></span>
	</div>
	<el-ctable :limit="4" class="tableCard" ref="cardtable" :readonly="selectionShow" :default-checked="deviceChecked" @select="checkedBox" :type="tbType" @card-menu-click="clickMenu" :card-option="cardOptions" :id="'alarmCardTempList'" row-key="template_id" :rownumber="true" :height="'100%'" style="margin: 0 15px;" :url="tbURL" :pagination="true" :query-params="queryParams">
		<template slot="toolbar">
			<div class="search-container">
				<div class="queryGroup" style="margin-left: 15px;">
					<input v-model="queryForm.searchText" placeholder="<%=rb.getString("MuBanMingCheng")%>" />
					<b class="el-icon el-icon-common-search" @click="query"></b>
				</div>
				<div class="tool-container">
					<div class="icon-box-div" @click="tbType=='card'?tbType='':tbType='card'"><i class="el-icon el-icon-table"></i></div>
				</div>
			</div>
			<div class="selected-tmp-con">
				<div class="select-box-container">
					<div class="select-box" v-for="(item,index) in selectTemArr" :key="index">
						<el-tooltip placement="top" :content="item.template_name">
							<span>{{item.template_name}}</span>
						</el-tooltip>
						
						<i class="el-icon el-icon-circle-close CODE_ALARM_VIEW hidden" style='font-size:18px;' @click="delSelectedTpl(item.template_id)"></i>
					</div>
					
				</div>
				<div>
					({{selectTemArr.length}}/4)<%=rb.getString("MuBanXianShi")%>
				</div>
				
			</div>
		</template>
		
		<el-table-column type="selection" width="60"></el-table-column>
		<el-table-column label='<%=rb.getString("CaoZuo")%>' width="90">
			<template slot-scope="scope">
            	<div class="operation_more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
          	</template>
		</el-table-column>
		<el-table-column sortable label='<%=rb.getString("MuBanMingCheng")%>' width="200" prop="template_name"></el-table-column>
		<el-table-column label='<%=rb.getString("GengXinShiJian")%>' prop="update_time"></el-table-column>
	</el-ctable>
	<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
	
	 <el-slide ref="slider" :url="slideUrl" :title="slideTitle" :footer="footerShow" :header='headerShow' :position="slidePosition" :modal="false" :height="sliderHeight" :width="sliderWidth" 
	  @ok='saveTemplate' @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 	
	 </el-slide>

</div>
<script type="text/javascript">
	new Vue({
		el: '#card_alarm_app',
		data(){
			return {
				deviceChecked:[],
				deviceCheckedBuf: [],
				tableDisable:false,
				importClass:'circleBg close_circle ',
				selectTemArr:[],
				tbType: 'card',
				tbURL: '${ctx}/fault/viewConfig/queryViewConfigPageList.action?timeZone='+timeZone,
				menus:[],
			    queryForm: {
			    	searchText: ''
			    },
			    queryParams: {
			    	searchText: '',
			    	timeZone: timeZone,
			    	order:'',
			    },
			    addText:false,
			    slidePosition: 'top',
			    slideTitle: '',
			    slideUrl: '',
			    sliderHeight: '100%',
			    sliderWidth: '100%',
			    footerShow: true,
			    headerShow: true,
				cardOptions: {
					fields: [ // 要展示的属性字段列表
						'template_name',// 第一个用于展示卡片标题 
						'update_time',
					],
					menuFmt: function(data,row){// 如果有返回值，则为新的菜单数据
						//return data
						var status = row.status,
    	    				isStopShow = status == 'on';

						var menus = [
							{label:'<%=rb.getString("XinXi")%>', code:'info', cls:"el-icon el-icon-operation-info"},
							{label:'<%=rb.getString("XiuGai")%>', code:'edit', cls:"el-icon el-icon-operation-edit CODE_ALARM_VIEW hidden"},
							{label:'<%=rb.getString("ShanChu")%>', code:'del', cls:"el-icon el-icon-operation-delete CODE_ALARM_VIEW hidden"}
						]

						return menus
					},
					beforeClick: function(menuRow, row){
					},
					checkable: true
				}
			}
		},
		computed:{
			selectionShow() {
				return writableMap['CODE_ALARM_VIEW'] !== true;
			},
		},
		watch:{},
		methods: {
			/**
			* 已启用模板删除
			* @param templateId{number}   模板id
			*/
			delSelectedTpl(templateId){
				 var vm = this
				 for(var i = 0;i < vm.selectTemArr.length;i++){
					 if(vm.selectTemArr[i].template_id == templateId){
						 vm.selectTemArr.splice(i,1)
					     
						 if(vm.selectTemArr.length < 4){
							 vm.tableDisable = false;
						 }
						 if(vm.selectTemArr.length == 0){
							 vm.$refs['cardtable'].clearSelection();
						 }
					 }
					 
				 }
				 vm.deviceChecked = vm.selectTemArr.map(function(item){
 					return item.template_id
 				})
 				 /* vm.$refs['cardtable'].refresh() */
			},
			/**
			* 已启用模板删除
			* @param selection{Array}   勾选的集合
			* @param row{object}   行数据			
			*/
			checkedBox(selection,row){
				var vm = this;
				vm.selectTemArr = selection;
				//最多选择四个
				if(vm.selectTemArr.length >= 4){
					vm.tableDisable = true;
				}
			},
			// 右上关闭按钮，把当前选择模板提交
			closeViewTemple(){ 
				var vm = this;
				//将选择的模板传给后台 vm.selectTemArr 
				let templateIdArr = []
				templateIdArr = vm.selectTemArr.map(function(item,index){
					return item.template_id
				})
				//是否需要判断数组是否发生变化  与初始化的比较 
				//初始化默认勾选的数组deviceChecked 检查 templateIdArr中的每一项是否在deviceChecked中
				if(vm.deviceCheckedBuf.length == templateIdArr.length){//判断数组长度  长度不一样 肯定改变了
					var deviceCheckedSet = new Set(vm.deviceCheckedBuf);
					var templateIdArrSet = new Set(templateIdArr);
					var subset = []; //差集
					for(let item of deviceCheckedSet){
						if(!templateIdArrSet.has(item)){
							subset.push(item);
						}
					}
					//取差集 如果不为空则说明改变了
					if(subset.length == 0) {
						eventBus.$emit("close-view");
						return false;
					}
					
				}
				let templateId = templateIdArr.join(",");//传给后台的参数
				let params = {
						templateId
				}
				axios.post("${ctx}/fault/view/setViewTemplate.action",stringify(params)).then(function(response){
					if(response.data.success){
						eventBus.$emit("close-view");
					}else{
						vm.$message.error(response.data["message"]) 
					}
					
	 			})
				
			},
			//默认初始化
			initCardTable(){
				var vm = this
				axios.post("${ctx}/fault/view/getViewTemplateList.action").then(function(response){
	 				var data = response.data;
	 				vm.selectTemArr = data;
	 				//筛选出数据作为默认选中项
	 				vm.deviceChecked = vm.selectTemArr.map(function(item){
	 					return item.template_id
	 				})
	 				vm.deviceCheckedBuf = vm.selectTemArr.map(function(item){
	 					return item.template_id
	 				})
	 			})
	 			//判断能否勾选
	 			let selectNum = vm.$refs['cardtable'].getChecked.length;
				if(selectNum >= 4){
					vm.tableDisable = false;
				}else{
					vm.tableDisable = true;
				}
	 				
	 			
			},
			// 模板查询
			query(){ 
				var vm = this;
				Object.assign(vm.queryParams, vm.queryForm);
				vm.$refs['cardtable'].refresh()
			},
			/**
			* 表格点击更多事件
			* @param row{object}   行数据
			* @param ev{object}    event数据			
			*/
			optClick(row,ev){
    	    	var status = row.status;
		    	this.menus= [
			          {label:'<%=rb.getString("XinXi")%>', code:'info', cls:"el-icon el-icon-operation-info",row:row},
			          {label:'<%=rb.getString("XiuGai")%>', code:'edit', cls:"el-icon el-icon-operation-edit  CODE_ALARM_VIEW hidden",row:row},
			          {label:'<%=rb.getString("ShanChu")%>', code:'del', cls:"el-icon el-icon-operation-delete  CODE_ALARM_VIEW hidden",row:row}
			    ]
		    	var vm = this;
		    	this.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.menu.show(ev);
		    	});
    	    },
			/**
			* 模板查看页面
			* @param templateId{number}   模板id
			*/
    	    viewInfo(templateId){
    	    	var vm = this;
    	    	vm.slideUrl = '${ctx}/fault/viewConfig/goAddViewConfigPage.action?type=view';
    	    	vm.slideTitle = '<%=rb.getString("XinXi")%>';
    	    	vm.slidePosition = 'right';
    	    	vm.sliderHeight = '100%';
    	    	vm.sliderWidth = '75%';
    	    	vm.footerShow = false;
    	    	vm.headerShow = true;
    	    	vm.$refs.slider.showSlide(function(){
    	    		eventBus.$emit('action-init',templateId,timeZone,'view');
    	    	});
    	    },
			/**
			* 模板修改页面
			* @param templateId{number}   模板id
			*/
    	    editAlarmTemp(templateId){
    	    	var vm = this;
    	    	vm.slideUrl = '${ctx}/fault/viewConfig/goAddViewConfigPage.action?type=edit';
    	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
    	    	vm.slidePosition = 'left';
    	    	vm.sliderHeight = '100%';
    	    	vm.sliderWidth = '75%';
    	    	vm.footerShow = true;
    	    	vm.headerShow = true;
    	    	vm.$refs.slider.showSlide(function(){
    	    		eventBus.$emit('action-init',templateId,timeZone,'edit');
    	    	});
    	    },
			// 点击页面其他地方菜单收起
			handerClose(){
    	        this.$refs.menu.hide();
    	    },
			/**
			* 菜单点击方法
			* @param ev{object}    card点击事件ev为card数据，表格点击事件时ev为行数据
			* @param row{object}   card点击事件row为card数据，表格点击事件时row为event数据
			*/
    	    clickMenu(ev,row){
    	    	var codes = {
    	    		info: this.viewInfo,
    	    		edit: this.editAlarmTemp,
    	    		del: this.delAlarmTemp
    	    	}
    	    	if(codes[ev.code]){
    	    		codes[ev.code](row.template_id || ev.row.template_id);
    	    	}
    	    },
			/**
			* 模板删除事件
			* @param templateId{number}    模板id
			*/
    	    delAlarmTemp(templateId) {
    	    	var vm = this,
		    		params = {
	    	    		templateId: templateId
	    	    	};
    	    	//判断当前Id 是否是被选中的模板  是：删除相关数据
    	    	//vm.selectTemArr 
    	    	vm.$confirm('<%=rb.getString("QueRenShanChuMuBan")%>','<%=rb.getString("QueRen")%>').then(function(){
    	    		vm.selectTemArr.forEach(function(item,index){
        	    		if(item.template_id == templateId){
        	    			vm.selectTemArr.splice(index,1)
        	    		}
        	    	})
    	    		axios.post('${ctx}/fault/viewConfig/deleteViewConfig.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data) {
							if(data["success"]){
		    					vm.$message({
		    						message: '<%=rb.getString("ChengGong")%>',
		    						type:'success'
		    					});
		    					vm.$refs["cardtable"].refresh()
		    					
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						}
					}).catch(function(error){})
    	    	});
    	    },
			// 触发新增或修改界面的保存操作
    	    saveTemplate(){
    	    	eventBus.$emit('action-save');
    	    },
			// 关闭侧滑页
    	    cancelSlide(){
    	    	this.$refs.slider.hide();
    	    },
			// 响应新增或修改界面保存成功处理
    	    saveOK(){
    	    	this.cancelSlide();
    	    },
			// 刷新
    	    reFreshTable(){
    	    	this.$refs["cardtable"].refresh()
    	    }
		},
		components:{},
		mounted(){
			this.initCardTable();
			eventBus.$off('refresh-card').$on('refresh-card',this.reFreshTable);
			eventBus.$off('action-ok').$on('action-ok',this.saveOK);
			eventBus.$off('action-cancel').$on('action-cancel',this.cancelSlide);
		}
	});
</script>