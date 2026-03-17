<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.el-badge{
	    position:relative;
    }
    .el-badge__content{
        position:absolute;
        top:6px;
        right:0px;
        transform:translateY(-50%) translateX(100%);
        background-color:transparent;
        border-radius:10px;
        color:#fff;
        display:inline-block;
        font-size:10px;
        height:12px;
        line-height:11px;
        padding:0 6px;
        text-align:center;
        white-space:nowrap;
        cursor:default;
        border:1px solid transparent;
    }
    .el-icon-star-badge:before{
        color:#F3916C;
    }
	.container .cmenu{
		z-index: 361!important;
	}
	#upsUpgradeSlide{
		z-index: 2002!important;
	}
</style>
<div class="pageDefault" id='upgrade_ups_ctn' style="overflow: auto;">
	<div class="container" style="min-width: 900px;">
        <div class="el-icon-copy-document"></div>
        <el-tabs class="fit">
          <el-tab-pane :label="upsUpgradeFileTitle">
            <!-- 表格组件 -->
            <el-ctable
                :url="upsUpgradeTableUrl" 
                :query-params="params" 
                ref="upsUpgradeTable" 
				:height="height" 
				:page-size="pageSize" 
				:page-list="pageList" 
				pagination="true">
                	<!-- 列表toolbar -->
                <template slot="toolbar">
                	<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
						
						<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:92px;top:5px;" @click="importUpgradeFile" tip="<%=rb.getString("DaoRuWenJian")%>">
							<span class='el-icon el-icon-circle-import'></span>
						</div>
						<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:56px;top:5px;" @click="upsUpgradeAddTask('','addTask')" tip="<%=rb.getString("XinJianRenWu")%>">
							<span class='el-icon el-icon-circle-addTask'></span>
						</div>
						<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="upsUpgradeTaskList" tip="<%=rb.getString("RenWuLieBiao")%>">
							<span class='el-icon el-icon-circle-taskList'></span>
						</div>
						
						<el-query type="normal" @query="queryUpsUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
					</div>
                </template>
                	<!-- 列表columns -->
                <el-table-column prop="op" label=" " width="40" align="center">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="showMenus(scope.row,event)"></div>
                    </template>
                </el-table-column>
                <el-table-column prop="version" label="<%=rb.getString("BanBen")%>" min-width="250">
                    <template slot-scope="scope">
                        <div class='el-badge'><span>{{scope.row.version}}</span><span v-show="scope.row.recommend == '1'" class='el-badge__content el-icon el-icon-star-badge'></span></div>
                    </template>
                </el-table-column>
                <el-table-column prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" min-width="150"></el-table-column>
                <el-table-column prop="size" label="<%=rb.getString("WenJianDaXiao")%>" min-width="120"></el-table-column>
                <el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" min-width="140"></el-table-column>
               
            </el-ctable>
          </el-tab-pane>
        </el-tabs>

        <!-- 菜单 -->
        <el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>
		<!-- slide -->
		<el-slide ref="upsUpgradeSlide" id="upsUpgradeSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		<el-slide ref="upsUpgradeResultSlide" id="upsUpgradeResultSlide" :url='resultSlideUrl' :title="resultSlideTitle" :footer="resultSlideFooter" :header="resultSlideHeader" :position="resultSlidePosition"
			:height="resultSlideHeight"  :width='resultSlideWidth' @cancel="resultSlideClose">
		</el-slide>
    </div>
</div>
<script type="text/javascript">
new Vue({
	el:'#upgrade_ups_ctn',
	data(){
		return {
            params:{
                timeZone:timeZone,
                searchText:'',
            },
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',

			resultSlideUrl:'',
			resultSlideTitle:'',
			resultSlideHeader:'',
			resultSlideFooter:'',
			resultSlidePosition:'',
			resultSlideHeight:'',
			resultSlideWidth:'',
            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
            menus:[],
            rowData:'',
            upsUpgradeTableUrl:'${ctx}/cell/version/queryfileInfosList.action?file_type=5',
			slideOpenType:'',

		}
	},
    computed:{
        upsUpgradeFileTitle(){
            return 'UPS' + ' ' + '<%=rb.getString("ShengJi")%>' + ' ' + '<%=rb.getString("WenJian")%>'
        },
		optBtnShow() {
			return writableMap['CODE_UPS'] == true;
		},
    },
	methods:{
		
        hideMenus() {
            this.$refs.menu.hide()
        },
        showMenus(row,evt) {
            var vm = this,recommendFlag="";

            vm.rowData = row;
			if(row.recommend == '1'){//说明此文件是推荐文件
				recommendFlag = false;
			}else{
				recommendFlag = true;
			}
            vm.menus = [
                {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
                {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download CODE_UPS hidden",code:'download'},
                {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_UPS hidden",code:'modify'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_UPS hidden",code:'del'},
                {label:'<%=rb.getString("TuiJian")%>',cls:"el-icon el-icon-operation-recommend CODE_UPS hidden",code:'recommend',show:recommendFlag},
                {label:'<%=rb.getString("QuXiaoTuiJian")%>',cls:"el-icon el-icon-operation-cancel-recommend CODE_UPS hidden",code:'recommend',show:!recommendFlag},
            ];
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.menu.show(evt)
            })
        },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
        menuClick(evt) {
			var vm = this;

			var codes = {
	    		view:this.upsUpgradeInfo,  // 详情
	    		download:this.upsUpgradeDownload,	// 下载
	    		modify:this.upsUpgradeModify,	// 修改
	    		del:this.upsUpgradeDel,	// 删除
	    		recommend:this.upsUpgradeRecommend,	// 推荐  取消推荐
	    	},
			recommend = '',
			fileType = 5,
			fileName = vm.rowData.file_name;
			if(vm.rowData.recommend == '1'){//说明此文件是推荐文件
				recommend = '0';
			}else{
				recommend = '1';
			}
			if(codes[evt.code]) {
				if(evt.code == 'download'){
					codes[evt.code](fileName,fileType);
				}
				else if(evt.code == 'recommend'){
					codes[evt.code](vm.rowData.id,recommend)
				}else{
					codes[evt.code](vm.rowData.id);
				} 
			}
        },
		/**
		 * 查看文件
		 * @param id:当前数据id
		*/
		upsUpgradeInfo(id){
			var vm = this;
			vm.slideOpenType = 'view';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/ups/toFileEdit.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.upsUpgradeSlide.showSlide(()=>{
				eventBus.$emit('ups-upgrade-editInt',id,'view');
			})
			event.stopPropagation();// 禁止事件穿透
		},
		/**
		 * 下载文件
		 * @param fileName:文件名称
		 * @param fileType: 类型
		*/
		upsUpgradeDownload(fileName,fileType){
			var vm = this;
			axios.post('${ctx}/omc/version/file/fileIsExist.action',stringify({
	    		fileName : fileName,
	    		fileType:fileType
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			exportByForm("${ctx}/omc/version/file/downLoadFile.action",{
						fileName: fileName,
						fileType: fileType
					});
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 修改文件
		 * @param id:当前数据id
		*/
		upsUpgradeModify(id){
			var vm = this;
			vm.slideOpenType = 'edit';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/ups/toFileEdit.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.upsUpgradeSlide.showSlide(()=>{
				eventBus.$emit('ups-upgrade-editInt',id,'edit');
			})
		},
		/**
		 * 删除文件
		 * @param id:当前数据id
		*/
		upsUpgradeDel(id){
			var vm = this,
				params = {
					fileID: id
				};
			vm.$confirm('<%=rb.getString("QueDingShanChuWenJian")%>','<%=rb.getString("QueRen")%>').then(function(){
				axios.post('${ctx}/cell/version/deleteVersionFile.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.upsUpgradeTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		/**
		 * 推荐
		 * @param id:当前数据id
		 * @param recommend:推荐状态  0：未推荐  1：已推荐
		*/
		upsUpgradeRecommend(id,recommend){
			var vm = this,
				params = {
					id: id,
					recommend:recommend
				};
			axios.post('${ctx}/cell/version/updateRecommendStatus.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$refs.upsUpgradeTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
		},
		// slide 提交
		submitSlide(){
			var vm = this;
			if(vm.slideOpenType == 'edit'){
				eventBus.$emit('ups-upgrade-editSubmit');
			}else if(vm.slideOpenType == 'import'){
				eventBus.$emit('ups-upgrade-importSubmit');
			}else if(vm.slideOpenType == 'addTask' || vm.slideOpenType == 'taskEdit'){
				eventBus.$emit('ups-upgrade-addSubmit');
			}
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
          
			vm.$refs.upsUpgradeSlide.hide();
			vm.$refs.upsUpgradeTable.refresh();
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
            
			if(vm.slideOpenType == 'edit'){
				eventBus.$emit('ups-upgrade-cancelEdit');
			}else if(vm.slideOpenType == 'import'){
				eventBus.$emit('ups-upgrade-cancelImport');
			}else if(vm.slideOpenType == 'addTask' || vm.slideOpenType == 'taskEdit'){
				eventBus.$emit('ups-upgrade-addCancel');
			}else{
				vm.$refs.upsUpgradeSlide.hide();
			}
			vm.$refs.upsUpgradeTable.refresh();
		},
        // UPS升级文件搜索事件
        queryUpsUpgradeFile(val){
            var vm = this;
            vm.params.searchText = val;
        },
		// 导入文件按钮
        importUpgradeFile(){
			var vm = this;
			vm.slideOpenType = 'import';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/ups/toImportFile.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("DaoRuWenJian")%>';
			vm.$refs.upsUpgradeSlide.showSlide(()=>{
				
			})
		},
		// 新建升级任务按钮
		upsUpgradeAddTask(id,type){
			var vm = this;
			vm.slideOpenType = type;
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/ups/toAddTask.action';
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			if(type == 'addTask'){
				vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
			}else if(type == 'taskEdit'){
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			}else if(type == 'taskView'){
				vm.slideTitle = '<%=rb.getString("ChaKan")%>';
			}
			if(type !== 'taskView'){
				vm.slideFooter = true;
			}else{
				vm.slideFooter = false;
			}
			vm.$refs.upsUpgradeSlide.showSlide(()=>{
				eventBus.$emit('ups-upgrade-taskInit',id,type);
			})
		},
		// 升级任务列表页
		upsUpgradeTaskList(){
			var vm = this;
			vm.slideOpenType = 'taskList';
			vm.resultSlideHeader = true;
			vm.resultSlideUrl = '${ctx}/task/upgrade/ups/toTaskList.action';
			vm.resultSlideFooter = false;
			vm.resultSlidePosition = 'top';
			vm.resultSlideHeight = '100%';
			vm.resultSlideWidth = '100%';
			vm.resultSlideTitle = '<%=rb.getString("RenWuLieBiao")%>';
			vm.$refs.upsUpgradeResultSlide.showSlide(()=>{
				
			})
		},
		// 任务列表页关闭
		resultSlideClose(){
			var vm = this;
			vm.$refs.upsUpgradeResultSlide.hide();
		},


	},
	mounted(){
		eventBus.$off('hide-upsUpgrade-slide').$on('hide-upsUpgrade-slide',this.hideSlide);
		eventBus.$off('open-upsUpgrade-addTask').$on('open-upsUpgrade-addTask',this.upsUpgradeAddTask);
	}
	
})

</script> 
